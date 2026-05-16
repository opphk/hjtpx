package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type ClusterState string

const (
	ClusterStateForming   ClusterState = "forming"
	ClusterStateStable    ClusterState = "stable"
	ClusterStateDegraded ClusterState = "degraded"
	ClusterStateSplitBrain ClusterState = "split_brain"
	ClusterStateOffline   ClusterState = "offline"
)

type ClusterConfig struct {
	ClusterName      string
	NodeID          string
	BindAddress     string
	AdvertiseAddress string
	Port            int
	GossipPort      int
	InitialNodes    []string
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	ElectionTimeout   time.Duration
	RetryInterval     time.Duration
	MaxRetries        int
	EnableEncryption  bool
	EncryptionKey     []byte
}

func DefaultClusterConfig(nodeID string) *ClusterConfig {
	return &ClusterConfig{
		ClusterName:       "hjtpx-cluster",
		NodeID:            nodeID,
		HeartbeatInterval: 1 * time.Second,
		HeartbeatTimeout:  3 * time.Second,
		ElectionTimeout:   5 * time.Second,
		RetryInterval:     1 * time.Second,
		MaxRetries:        5,
		EnableEncryption:  false,
	}
}

type ClusterMember struct {
	NodeID           string
	Address          string
	AdvertiseAddress string
	Port             int
	Status           MemberStatus
	Role             MemberRole
	Metadata         map[string]interface{}
	Votes            int
	LastHeartbeat    time.Time
	StartTime        time.Time
	mu               sync.RWMutex
}

type MemberStatus string

const (
	MemberStatusAlive   MemberStatus = "alive"
	MemberStatusSuspect MemberStatus = "suspect"
	MemberStatusDead   MemberStatus = "dead"
	MemberStatusLeft   MemberStatus = "left"
)

type MemberRole string

const (
	RoleLeader    MemberRole = "leader"
	RoleFollower  MemberRole = "follower"
	RoleCandidate MemberRole = "candidate"
	RoleStandby   MemberRole = "standby"
)

type ClusterManager struct {
	config         *ClusterConfig
	state          atomic.Value
	members        map[string]*ClusterMember
	leaderNodeID   atomic.Value
	currentRole    atomic.Value
	memberMu       sync.RWMutex
	httpServer     *http.Server
	gossipServer   *GossipServer
	raftNode       *RaftNode
	notifyChan     chan *ClusterEvent
	stopChan       chan struct{}
	wg             sync.WaitGroup
	version        int64
	term           int64
	votedFor       atomic.Value
	lastLogIndex   atomic.Int64
	lastLogTerm    atomic.Int64
	commitIndex    atomic.Int64
	lastApplied    atomic.Int64
	log            []LogEntry
	logMu          sync.RWMutex
	eventHandlers  []ClusterEventHandler
}

type LogEntry struct {
	Term         int64
	Index        int64
	CommandType  string
	Command      []byte
	ClientAddr   string
	Timestamp    time.Time
}

type ClusterEvent struct {
	Type      ClusterEventType
	NodeID    string
	Timestamp time.Time
	Member    *ClusterMember
	Metadata  map[string]interface{}
	Error     error
}

type ClusterEventType string

const (
	ClusterEventMemberJoined    ClusterEventType = "member_joined"
	ClusterEventMemberLeft      ClusterEventType = "member_left"
	ClusterEventMemberFailed    ClusterEventType = "member_failed"
	ClusterEventMemberRecovered ClusterEventType = "member_recovered"
	ClusterEventLeaderElected   ClusterEventType = "leader_elected"
	ClusterEventStateChange     ClusterEventType = "state_change"
	ClusterEventConfigChange    ClusterEventType = "config_change"
	ClusterEventSplitBrain      ClusterEventType = "split_brain"
	ClusterEventRecovery        ClusterEventType = "recovery"
)

type ClusterEventHandler func(event *ClusterEvent)

type GossipServer struct {
	BindAddress string
	Port       int
	cluster    *ClusterManager
	quit       chan struct{}
}

type RaftNode struct {
	nodeID       string
	cluster      *ClusterManager
	currentState RaftState
	mu           sync.RWMutex
	heartbeatT   *time.Timer
	electionT    *time.Timer
}

type RaftState string

const (
	RaftStateFollower  RaftState = "follower"
	RaftStateCandidate RaftState = "candidate"
	RaftStateLeader    RaftState = "leader"
)

func NewClusterManager(config *ClusterConfig) (*ClusterManager, error) {
	if config == nil {
		config = DefaultClusterConfig("node-" + generateNodeID())
	}

	cm := &ClusterManager{
		config:      config,
		members:     make(map[string]*ClusterMember),
		notifyChan:  make(chan *ClusterEvent, 100),
		stopChan:    make(chan struct{}),
		log:        make([]LogEntry, 0),
	}

	cm.state.Store(ClusterStateForming)
	cm.currentRole.Store(RoleFollower)
	cm.votedFor.Store("")

	cm.addSelfAsMember()

	return cm, nil
}

func generateNodeID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix()%10000)
}

func (cm *ClusterManager) addSelfAsMember() {
	cm.memberMu.Lock()
	defer cm.memberMu.Unlock()

	member := &ClusterMember{
		NodeID:        cm.config.NodeID,
		Address:       cm.config.BindAddress,
		AdvertiseAddress: cm.config.AdvertiseAddress,
		Port:          cm.config.Port,
		Status:        MemberStatusAlive,
		Role:          RoleFollower,
		Metadata:      make(map[string]interface{}),
		StartTime:     time.Now(),
		LastHeartbeat: time.Now(),
	}

	cm.members[cm.config.NodeID] = member
}

func (cm *ClusterManager) Start(ctx context.Context) error {
	cm.gossipServer = &GossipServer{
		BindAddress: cm.config.BindAddress,
		Port:        cm.config.GossipPort,
		cluster:     cm,
		quit:        make(chan struct{}),
	}

	cm.raftNode = &RaftNode{
		nodeID:       cm.config.NodeID,
		cluster:      cm,
		currentState: RaftStateFollower,
	}

	cm.wg.Add(1)
	go cm.gossipServer.serve()

	cm.wg.Add(1)
	go cm.raftNode.run()

	cm.wg.Add(1)
	go cm.monitorMembers()

	cm.wg.Add(1)
	go cm.processEvents()

	for _, nodeAddr := range cm.config.InitialNodes {
		if err := cm.joinNode(nodeAddr); err != nil {
			continue
		}
	}

	cm.setState(ClusterStateStable)

	return nil
}

func (cm *ClusterManager) Stop() {
	close(cm.stopChan)

	if cm.gossipServer != nil {
		close(cm.gossipServer.quit)
	}

	cm.wg.Wait()

	cm.setState(ClusterStateOffline)
}

func (cm *ClusterManager) setState(state ClusterState) {
	oldState := cm.state.Load().(ClusterState)
	cm.state.Store(state)

	if oldState != state {
		cm.notify(&ClusterEvent{
			Type:      ClusterEventStateChange,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"old_state": oldState,
				"new_state": state,
			},
		})
	}
}

func (cm *ClusterManager) GetState() ClusterState {
	return cm.state.Load().(ClusterState)
}

func (cm *ClusterManager) Join(address string) error {
	return cm.joinNode(address)
}

func (cm *ClusterManager) joinNode(address string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/cluster/join", address), nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Node-ID", cm.config.NodeID)
	req.Header.Set("X-Node-Addr", cm.config.AdvertiseAddress)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("join failed with status %d", resp.StatusCode)
	}

	var joinResp struct {
		NodeID   string   `json:"node_id"`
		Members  []string `json:"members"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&joinResp); err != nil {
		return err
	}

	cm.notify(&ClusterEvent{
		Type:      ClusterEventMemberJoined,
		NodeID:    joinResp.NodeID,
		Timestamp: time.Now(),
	})

	return nil
}

func (cm *ClusterManager) Leave(nodeID string) error {
	cm.memberMu.Lock()
	defer cm.memberMu.Unlock()

	if member, exists := cm.members[nodeID]; exists {
		member.Status = MemberStatusLeft
		delete(cm.members, nodeID)

		cm.notify(&ClusterEvent{
			Type:      ClusterEventMemberLeft,
			NodeID:    nodeID,
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (cm *ClusterManager) GetMembers() []*ClusterMember {
	cm.memberMu.RLock()
	defer cm.memberMu.RUnlock()

	members := make([]*ClusterMember, 0, len(cm.members))
	for _, member := range cm.members {
		member.mu.RLock()
		members = append(members, &ClusterMember{
			NodeID:           member.NodeID,
			Address:          member.Address,
			AdvertiseAddress: member.AdvertiseAddress,
			Port:             member.Port,
			Status:           member.Status,
			Role:             member.Role,
			Metadata:         member.Metadata,
			Votes:            member.Votes,
			LastHeartbeat:    member.LastHeartbeat,
			StartTime:        member.StartTime,
		})
		member.mu.RUnlock()
	}

	return members
}

func (cm *ClusterManager) GetLeader() (string, error) {
	leader := cm.leaderNodeID.Load()
	if leader == nil {
		return "", fmt.Errorf("no leader elected")
	}
	return leader.(string), nil
}

func (cm *ClusterManager) IsLeader() bool {
	return cm.currentRole.Load().(MemberRole) == RoleLeader
}

func (cm *ClusterManager) GetCurrentRole() MemberRole {
	return cm.currentRole.Load().(MemberRole)
}

func (cm *ClusterManager) SetLeader(nodeID string) {
	cm.leaderNodeID.Store(nodeID)

	if nodeID == cm.config.NodeID {
		cm.currentRole.Store(RoleLeader)
		cm.updateMemberRole(cm.config.NodeID, RoleLeader)
	} else {
		cm.updateMemberRole(nodeID, RoleLeader)
	}

	cm.notify(&ClusterEvent{
		Type:      ClusterEventLeaderElected,
		NodeID:    nodeID,
		Timestamp: time.Now(),
	})
}

func (cm *ClusterManager) updateMemberRole(nodeID string, role MemberRole) {
	cm.memberMu.Lock()
	defer cm.memberMu.Unlock()

	if member, exists := cm.members[nodeID]; exists {
		member.mu.Lock()
		member.Role = role
		member.mu.Unlock()
	}
}

func (cm *ClusterManager) monitorMembers() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopChan:
			return
		case <-ticker.C:
			cm.checkMemberHealth()
		}
	}
}

func (cm *ClusterManager) checkMemberHealth() {
	cm.memberMu.Lock()
	defer cm.memberMu.Unlock()

	now := time.Now()
	for nodeID, member := range cm.members {
		if nodeID == cm.config.NodeID {
			continue
		}

		member.mu.Lock()
		timeSinceHeartbeat := now.Sub(member.LastHeartbeat)
		previousStatus := member.Status

		if timeSinceHeartbeat > cm.config.HeartbeatTimeout {
			if member.Status == MemberStatusAlive {
				member.Status = MemberStatusSuspect
			} else if member.Status == MemberStatusSuspect {
				member.Status = MemberStatusDead
			}
		} else if member.Status == MemberStatusSuspect {
			member.Status = MemberStatusAlive
		}
		member.mu.Unlock()

		if previousStatus != member.Status {
			if member.Status == MemberStatusDead {
				cm.notify(&ClusterEvent{
					Type:      ClusterEventMemberFailed,
					NodeID:    nodeID,
					Timestamp: now,
				})
			} else if member.Status == MemberStatusAlive && previousStatus == MemberStatusSuspect {
				cm.notify(&ClusterEvent{
					Type:      ClusterEventMemberRecovered,
					NodeID:    nodeID,
					Timestamp: now,
				})
			}
		}
	}

	cm.updateClusterState()
}

func (cm *ClusterManager) updateClusterState() {
	cm.memberMu.RLock()
	defer cm.memberMu.RUnlock()

	var aliveCount, deadCount, totalCount int
	for _, member := range cm.members {
		totalCount++
		if member.Status == MemberStatusAlive {
			aliveCount++
		} else if member.Status == MemberStatusDead {
			deadCount++
		}
	}

	if aliveCount == 0 {
		cm.setState(ClusterStateOffline)
	} else if deadCount > 0 && aliveCount > 0 {
		cm.setState(ClusterStateDegraded)
	} else if cm.isSplitBrain() {
		cm.setState(ClusterStateSplitBrain)
	} else {
		cm.setState(ClusterStateStable)
	}
}

func (cm *ClusterManager) isSplitBrain() bool {
	cm.memberMu.RLock()
	defer cm.memberMu.RUnlock()

	var leaders []string
	for _, member := range cm.members {
		if member.Status == MemberStatusAlive && member.Role == RoleLeader {
			leaders = append(leaders, member.NodeID)
		}
	}

	return len(leaders) > 1
}

func (cm *ClusterManager) processEvents() {
	defer cm.wg.Done()

	for {
		select {
		case <-cm.stopChan:
			return
		case event := <-cm.notifyChan:
			cm.handleEvent(event)
		}
	}
}

func (cm *ClusterManager) handleEvent(event *ClusterEvent) {
	for _, handler := range cm.eventHandlers {
		go handler(event)
	}
}

func (cm *ClusterManager) notify(event *ClusterEvent) {
	select {
	case cm.notifyChan <- event:
	default:
	}
}

func (cm *ClusterManager) AddEventHandler(handler ClusterEventHandler) {
	cm.eventHandlers = append(cm.eventHandlers, handler)
}

func (cm *ClusterManager) SubmitCommand(commandType string, data []byte) (int64, error) {
	cm.logMu.Lock()
	defer cm.logMu.Unlock()

	cm.lastLogIndex.Add(1)
	index := cm.lastLogIndex.Load()

	entry := LogEntry{
		Term:    atomic.LoadInt64(&cm.term),
		Index:   index,
		CommandType: commandType,
		Command: data,
		Timestamp: time.Now(),
	}

	cm.log = append(cm.log, entry)

	return index, nil
}

func (cm *ClusterManager) GetLastLogIndex() int64 {
	return cm.lastLogIndex.Load()
}

func (cm *ClusterManager) GetLastLogTerm() int64 {
	return cm.lastLogTerm.Load()
}

func (cm *ClusterManager) GetLogEntries(desdeIndex int64) ([]LogEntry, error) {
	cm.logMu.RLock()
	defer cm.logMu.RUnlock()

	var entries []LogEntry
	for _, entry := range cm.log {
		if entry.Index >= desdeIndex {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (gs *GossipServer) serve() {
	defer gs.cluster.wg.Done()

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", gs.BindAddress, gs.Port))
	if err != nil {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/gossip", gs.handleGossip)
	mux.HandleFunc("/cluster/join", gs.handleJoin)

	server := &http.Server{
		Handler: mux,
	}

	go server.Serve(ln)

	<-gs.quit
	server.Close()
}

func (gs *GossipServer) handleGossip(w http.ResponseWriter, r *http.Request) {
	var gossipData struct {
		NodeID        string                 `json:"node_id"`
		Members       map[string]interface{}  `json:"members"`
		State         string                 `json:"state"`
		Term          int64                  `json:"term"`
		LastLogIndex  int64                  `json:"last_log_index"`
	}

	if err := json.NewDecoder(r.Body).Decode(&gossipData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gs.cluster.memberMu.Lock()
	if member, exists := gs.cluster.members[gossipData.NodeID]; exists {
		member.mu.Lock()
		member.Status = MemberStatusAlive
		member.LastHeartbeat = time.Now()
		member.mu.Unlock()
	}
	gs.cluster.memberMu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (gs *GossipServer) handleJoin(w http.ResponseWriter, r *http.Request) {
	nodeID := r.Header.Get("X-Node-ID")
	nodeAddr := r.Header.Get("X-Node-Addr")

	if nodeID == "" || nodeAddr == "" {
		http.Error(w, "missing node info", http.StatusBadRequest)
		return
	}

	gs.cluster.memberMu.Lock()
	defer gs.cluster.memberMu.Unlock()

	if _, exists := gs.cluster.members[nodeID]; !exists {
		gs.cluster.members[nodeID] = &ClusterMember{
			NodeID:           nodeID,
			Address:          nodeAddr,
			Status:           MemberStatusAlive,
			StartTime:        time.Now(),
			LastHeartbeat:    time.Now(),
			Metadata:         make(map[string]interface{}),
		}
	}

	memberIDs := make([]string, 0, len(gs.cluster.members))
	for id := range gs.cluster.members {
		memberIDs = append(memberIDs, id)
	}

	resp := map[string]interface{}{
		"node_id":  gs.cluster.config.NodeID,
		"members":  memberIDs,
		"success": true,
	}

	json.NewEncoder(w).Encode(resp)
}

func (rn *RaftNode) run() {
	defer rn.cluster.wg.Done()

	rn.resetElectionTimer()

	for {
		select {
		case <-rn.cluster.stopChan:
			return
		default:
			rn.mu.Lock()
			state := rn.currentState
			rn.mu.Unlock()

			switch state {
			case RaftStateFollower:
				rn.runFollower()
			case RaftStateCandidate:
				rn.runCandidate()
			case RaftStateLeader:
				rn.runLeader()
			}
		}
	}
}

func (rn *RaftNode) runFollower() {
	<-rn.electionT.C
	rn.becomeCandidate()
}

func (rn *RaftNode) runCandidate() {
	rn.startElection()

	timeout := time.After(rn.cluster.config.ElectionTimeout)
	for {
		select {
		case <-rn.cluster.stopChan:
			return
		case <-timeout:
			rn.becomeCandidate()
			return
		case <-rn.heartbeatT.C:
			if rn.currentState == RaftStateCandidate {
				return
			}
		}
	}
}

func (rn *RaftNode) runLeader() {
	rn.sendHeartbeats()

	ticker := time.NewTicker(rn.cluster.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rn.cluster.stopChan:
			return
		case <-ticker.C:
			rn.sendHeartbeats()
		}
	}
}

func (rn *RaftNode) becomeCandidate() {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	rn.currentState = RaftStateCandidate
	atomic.AddInt64(&rn.cluster.term, 1)
	term := atomic.LoadInt64(&rn.cluster.term)
	rn.cluster.votedFor.Store(rn.cluster.config.NodeID)

	rn.resetElectionTimer()
}

func (rn *RaftNode) startElection() {
	rn.mu.RLock()
	isCandidate := rn.currentState == RaftStateCandidate
	rn.mu.RUnlock()

	if !isCandidate {
		return
	}

	term := atomic.LoadInt64(&rn.cluster.term)

	rn.cluster.memberMu.RLock()
	votes := 1
	for nodeID, member := range rn.cluster.members {
		if nodeID == rn.cluster.config.NodeID {
			continue
		}

		member.mu.RLock()
		if member.Status == MemberStatusAlive {
			votes++
		}
		member.mu.RUnlock()
	}
	rn.cluster.memberMu.RUnlock()

	if votes > len(rn.cluster.members)/2 {
		rn.becomeLeader()
	}
}

func (rn *RaftNode) becomeLeader() {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	rn.currentState = RaftStateLeader
	rn.cluster.SetLeader(rn.cluster.config.NodeID)
}

func (rn *RaftNode) resetElectionTimer() {
	if rn.electionT != nil {
		rn.electionT.Stop()
	}
	rn.electionT = time.NewTimer(rn.cluster.config.ElectionTimeout)
}

func (rn *RaftNode) sendHeartbeats() {
	rn.mu.RLock()
	isLeader := rn.currentState == RaftStateLeader
	rn.mu.RUnlock()

	if !isLeader {
		return
	}

	rn.cluster.memberMu.RLock()
	for nodeID, member := range rn.cluster.members {
		if nodeID == rn.cluster.config.NodeID {
			continue
		}

		member.mu.RLock()
		if member.Status == MemberStatusAlive {
			go rn.sendHeartbeat(nodeID, member.Address)
		}
		member.mu.RUnlock()
	}
	rn.cluster.memberMu.RUnlock()

	if rn.heartbeatT != nil {
		rn.heartbeatT.Reset(rn.cluster.config.HeartbeatInterval)
	}
}

func (rn *RaftNode) sendHeartbeat(nodeID, address string) {
	hb := map[string]interface{}{
		"term":           atomic.LoadInt64(&rn.cluster.term),
		"leader_id":      rn.cluster.config.NodeID,
		"prev_log_index": rn.cluster.lastLogIndex.Load(),
		"prev_log_term":  rn.cluster.lastLogTerm.Load(),
		"commit_index":  rn.cluster.commitIndex.Load(),
	}

	data, _ := json.Marshal(hb)
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://%s/raft/heartbeat", address), nil)
	req.Body = nil

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var reply struct {
		Term    int64  `json:"term"`
		Success bool   `json:"success"`
	}

	json.NewDecoder(resp.Body).Decode(&reply)

	if reply.Term > atomic.LoadInt64(&rn.cluster.term) {
		rn.becomeFollower(reply.Term)
	}
}

func (rn *RaftNode) becomeFollower(term int64) {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	rn.currentState = RaftStateFollower
	atomic.StoreInt64(&rn.cluster.term, term)
	rn.resetElectionTimer()
}

type ClusterStatus struct {
	State        ClusterState       `json:"state"`
	NodeID       string             `json:"node_id"`
	Role         MemberRole         `json:"role"`
	IsLeader     bool               `json:"is_leader"`
	Members      []*ClusterMember   `json:"members"`
	LeaderID     string             `json:"leader_id"`
	Term         int64              `json:"term"`
	CommitIndex  int64              `json:"commit_index"`
	LastLogIndex int64              `json:"last_log_index"`
}

func (cm *ClusterManager) GetStatus() *ClusterStatus {
	leaderID, _ := cm.GetLeader()

	return &ClusterStatus{
		State:        cm.GetState(),
		NodeID:       cm.config.NodeID,
		Role:         cm.GetCurrentRole(),
		IsLeader:     cm.IsLeader(),
		Members:      cm.GetMembers(),
		LeaderID:     leaderID,
		Term:         atomic.LoadInt64(&cm.term),
		CommitIndex:  cm.commitIndex.Load(),
		LastLogIndex: cm.lastLogIndex.Load(),
	}
}
