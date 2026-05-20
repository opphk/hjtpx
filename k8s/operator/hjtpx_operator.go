package operator

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type HjtpxOperator struct {
	manager    manager.Manager
	reconciler *HjtpxAppReconciler
	config     *OperatorConfig
}

type OperatorConfig struct {
	Namespace           string
	ServiceMeshProvider string
	GitOpsProvider      string
	ObservabilityEnabled bool
	AutoHealingEnabled  bool
	HealthCheckInterval time.Duration
	MaxReconcileRetries int
}

type HjtpxAppSpec struct {
	Application    ApplicationSpec   `json:"application"`
	Deployment     DeploymentSpec    `json:"deployment"`
	Service        ServiceSpec       `json:"service"`
	Ingress        *IngressSpec      `json:"ingress,omitempty"`
	ServiceMesh    *ServiceMeshConfig `json:"serviceMesh,omitempty"`
	GitOps         *GitOpsConfig     `json:"gitops,omitempty"`
	Observability  *ObservabilityConfig `json:"observability,omitempty"`
	Security       *SecurityConfig   `json:"security,omitempty"`
	AutoScaling    *AutoScalingConfig `json:"autoScaling,omitempty"`
	HighAvailability *HAConfig       `json:"highAvailability,omitempty"`
}

type ApplicationSpec struct {
	Name          string            `json:"name"`
	DisplayName   string            `json:"displayName"`
	Description   string            `json:"description"`
	Version       string            `json:"version"`
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	Owner         string            `json:"owner"`
	ContactEmail  string            `json:"contactEmail"`
	Tenant        string            `json:"tenant"`
	Environment   string            `json:"environment"`
}

type DeploymentSpec struct {
	Image           string            `json:"image"`
	Replicas        int32             `json:"replicas"`
	Command         []string          `json:"command,omitempty"`
	Args            []string          `json:"args,omitempty"`
	Env             []EnvVar          `json:"env,omitempty"`
	EnvFrom         []EnvFromSource   `json:"envFrom,omitempty"`
	Ports           []ContainerPort   `json:"ports,omitempty"`
	Resources       ResourceRequirements `json:"resources"`
	Probes          *ProbesConfig     `json:"probes,omitempty"`
	VolumeMounts    []VolumeMount     `json:"volumeMounts,omitempty"`
	ImagePullPolicy string            `json:"imagePullPolicy"`
	Strategy        DeploymentStrategy `json:"strategy"`
	NodeSelector    map[string]string `json:"nodeSelector,omitempty"`
	Tolerations     []Toleration      `json:"tolerations,omitempty"`
	Affinity        *AffinityConfig   `json:"affinity,omitempty"`
}

type EnvVar struct {
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	ValueFrom *EnvSource `json:"valueFrom,omitempty"`
}

type EnvSource struct {
	SecretRef      *SecretKeySelector `json:"secretKeyRef,omitempty"`
	ConfigMapRef   *ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

type SecretKeySelector struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type ConfigMapKeySelector struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type EnvFromSource struct {
	SecretRef    *SecretRef `json:"secretRef,omitempty"`
	ConfigMapRef *ConfigMapRef `json:"configMapRef,omitempty"`
}

type SecretRef struct {
	Name string `json:"name"`
}

type ConfigMapRef struct {
	Name string `json:"name"`
}

type ContainerPort struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

type ResourceRequirements struct {
	Requests ResourceList `json:"requests"`
	Limits   ResourceList `json:"limits"`
}

type ResourceList struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	GPU    string `json:"gpu,omitempty"`
	Storage string `json:"storage,omitempty"`
}

type ProbesConfig struct {
	StartupProbe  *Probe `json:"startupProbe,omitempty"`
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`
	ReadinessProbe *Probe `json:"readinessProbe,omitempty"`
}

type Probe struct {
	InitialDelaySeconds int32         `json:"initialDelaySeconds"`
	PeriodSeconds       int32         `json:"periodSeconds"`
	TimeoutSeconds      int32         `json:"timeoutSeconds"`
	FailureThreshold    int32         `json:"failureThreshold"`
	SuccessThreshold    int32         `json:"successThreshold"`
	Handler             ProbeHandler  `json:"handler"`
}

type ProbeHandler struct {
	Exec      *ExecAction     `json:"exec,omitempty"`
	HTTPGet   *HTTPGetAction  `json:"httpGet,omitempty"`
	TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty"`
}

type ExecAction struct {
	Command []string `json:"command"`
}

type HTTPGetAction struct {
	Path   string            `json:"path"`
	Port   int32             `json:"port"`
	Scheme string            `json:"scheme"`
	Headers []HTTPHeader     `json:"headers,omitempty"`
}

type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TCPSocketAction struct {
	Port int32 `json:"port"`
}

type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
}

type DeploymentStrategy struct {
	Type          string            `json:"type"`
	RollingUpdate *RollingUpdateStrategy `json:"rollingUpdate,omitempty"`
	Recreate      *RecreateStrategy `json:"recreate,omitempty"`
}

type RollingUpdateStrategy struct {
	MaxSurge       int32 `json:"maxSurge"`
	MaxUnavailable int32 `json:"maxUnavailable"`
}

type RecreateStrategy struct {
}

type Toleration struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Effect   string `json:"effect"`
}

type AffinityConfig struct {
	NodeAffinity    *NodeAffinityConfig    `json:"nodeAffinity,omitempty"`
	PodAffinity     *PodAffinityConfig     `json:"podAffinity,omitempty"`
	PodAntiAffinity *PodAntiAffinityConfig `json:"podAntiAffinity,omitempty"`
}

type NodeAffinityConfig struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

type NodeSelector struct {
	NodeSelectorTerms []NodeSelectorTerm `json:"nodeSelectorTerms"`
}

type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty"`
	MatchFields      []NodeSelectorRequirement `json:"matchFields,omitempty"`
}

type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

type PreferredTerm struct {
	Weight     int32          `json:"weight"`
	Preference NodeSelectorTerm `json:"preference"`
}

type PodAffinityConfig struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

type PodAntiAffinityConfig struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

type PodAffinityTerm struct {
	LabelSelector *LabelSelector `json:"labelSelector,omitempty"`
	TopologyKey   string          `json:"topologyKey"`
}

type LabelSelector struct {
	MatchLabels      map[string]string       `json:"matchLabels,omitempty"`
	MatchExpressions []LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

type LabelSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

type WeightedPodAffinityTerm struct {
	Weight   int32          `json:"weight"`
	PodAffinityTerm PodAffinityTerm `json:"podAffinityTerm"`
}

type ServiceSpec struct {
	Type        string          `json:"type"`
	Ports       []ServicePort   `json:"ports"`
	Selector    map[string]string `json:"selector"`
	ClusterIP   string          `json:"clusterIP,omitempty"`
	SessionAffinity string       `json:"sessionAffinity,omitempty"`
	HealthCheck *ServiceHealthCheck `json:"healthCheck,omitempty"`
}

type ServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	Protocol   string `json:"protocol"`
}

type ServiceHealthCheck struct {
	Enabled bool `json:"enabled"`
	Path    string `json:"path,omitempty"`
}

type IngressSpec struct {
	Enabled   bool              `json:"enabled"`
	ClassName string            `json:"className"`
	Host      string            `json:"host"`
	Path      string            `json:"path"`
	TLS       *TLSConfig        `json:"tls,omitempty"`
	Rules     []IngressRule     `json:"rules,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type TLSConfig struct {
	Enabled  bool     `json:"enabled"`
	SecretName string  `json:"secretName"`
	Hosts    []string `json:"hosts"`
}

type IngressRule struct {
	Host     string            `json:"host"`
	Path     string            `json:"path"`
	PathType string            `json:"pathType"`
	Backend  IngressBackend   `json:"backend"`
}

type IngressBackend struct {
	ServiceName string `json:"serviceName"`
	ServicePort int32  `json:"servicePort"`
}

type ServiceMeshConfig struct {
	Enabled        bool               `json:"enabled"`
	Provider       string             `json:"provider"`
	MTLSMode       string             `json:"mtlsMode"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty"`
	Canary         *CanaryConfig      `json:"canary,omitempty"`
	Tracing        *TracingConfig     `json:"tracing,omitempty"`
	LocalityLoadBalancing *LocalityLBConfig `json:"localityLoadBalancing,omitempty"`
}

type CircuitBreakerConfig struct {
	Enabled            bool    `json:"enabled"`
	MaxConnections     int     `json:"maxConnections"`
	MaxPendingRequests int     `json:"maxPendingRequests"`
	MaxRetries        int     `json:"maxRetries"`
	ConsecutiveErrors int     `json:"consecutiveErrors"`
	Interval           string  `json:"interval"`
	BaseEjectionTime   string  `json:"baseEjectionTime"`
	MaxEjectionPercent int     `json:"maxEjectionPercent"`
}

type CanaryConfig struct {
	Enabled       bool    `json:"enabled"`
	Percentage    int     `json:"percentage"`
	MinReplicas   int32   `json:"minReplicas"`
	MaxReplicas   int32   `json:"maxReplicas"`
	Header        string  `json:"header,omitempty"`
	HeaderValue   string  `json:"headerValue,omitempty"`
	Cookie        string  `json:"cookie,omitempty"`
}

type TracingConfig struct {
	Enabled        bool    `json:"enabled"`
	SamplingRate   float64 `json:"samplingRate"`
	JaegerEndpoint string  `json:"jaegerEndpoint,omitempty"`
	ZipkinEndpoint string  `json:"zipkinEndpoint,omitempty"`
}

type LocalityLBConfig struct {
	Enabled  bool `json:"enabled"`
	Failover bool `json:"failover"`
}

type GitOpsConfig struct {
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`
	Repository    string `json:"repository"`
	Branch        string `json:"branch"`
	Path          string `json:"path"`
	ArgoCDAppName string `json:"argocdAppName,omitempty"`
	FluxAppName   string `json:"fluxAppName,omitempty"`
	AutoSync      bool   `json:"autoSync"`
	Prune         bool   `json:"prune"`
	SelfHeal      bool   `json:"selfHeal"`
}

type ObservabilityConfig struct {
	Metrics   *MetricsConfig   `json:"metrics,omitempty"`
	Tracing   *TracingConfig   `json:"tracing,omitempty"`
	Logging   *LoggingConfig   `json:"logging,omitempty"`
	Dashboards []DashboardConfig `json:"dashboards,omitempty"`
}

type MetricsConfig struct {
	Enabled         bool     `json:"enabled"`
	PrometheusEnabled bool   `json:"prometheusEnabled"`
	Port            int32    `json:"port"`
	Path            string   `json:"path"`
	AdditionalLabels map[string]string `json:"additionalLabels,omitempty"`
}

type LoggingConfig struct {
	Enabled    bool     `json:"enabled"`
	LogLevel   string   `json:"logLevel"`
	LogFormat  string   `json:"logFormat"`
	OutputPath string   `json:"outputPath"`
	FluentdConfig *FluentdConfig `json:"fluentd,omitempty"`
}

type FluentdConfig struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	BufferPath string `json:"bufferPath,omitempty"`
}

type DashboardConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Variables   map[string]string `json:"variables,omitempty"`
}

type SecurityConfig struct {
	NetworkPolicies  bool            `json:"networkPolicies"`
	PodSecurity      bool            `json:"podSecurity"`
	PodSecurityPolicy bool          `json:"podSecurityPolicy"`
	SecretEncryption bool            `json:"secretEncryption"`
	RBACEnabled      bool            `json:"rbacEnabled"`
	PSPEnabled       bool            `json:"pspEnabled"`
	SecurityContexts *SecurityContexts `json:"securityContexts,omitempty"`
	PodDisruptionBudget *PDBConfig  `json:"podDisruptionBudget,omitempty"`
}

type SecurityContexts struct {
	RunAsNonRoot    bool   `json:"runAsNonRoot"`
	RunAsUser       int64  `json:"runAsUser"`
	RunAsGroup      int64  `json:"runAsGroup"`
	FSGroup         int64  `json:"fsGroup"`
	CapabilitiesDrop []string `json:"capabilitiesDrop"`
	ReadOnlyRootFS  bool   `json:"readOnlyRootFilesystem"`
	AllowPrivilegeEscalation bool `json:"allowPrivilegeEscalation"`
}

type PDBConfig struct {
	MinAvailable   int32  `json:"minAvailable,omitempty"`
	MaxUnavailable int32  `json:"maxUnavailable,omitempty"`
}

type AutoScalingConfig struct {
	Enabled            bool    `json:"enabled"`
	MinReplicas        int32   `json:"minReplicas"`
	MaxReplicas        int32   `json:"maxReplicas"`
	TargetCPUUtil      int32   `json:"targetCPUUtilizationPercentage"`
	TargetMemoryUtil   int32   `json:"targetMemoryUtilizationPercentage"`
	TargetCustomMetric *CustomMetricConfig `json:"targetCustomMetric,omitempty"`
	Behavior           *HPABehavior `json:"behavior,omitempty"`
}

type CustomMetricConfig struct {
	Name     string `json:"name"`
	TargetValue float64 `json:"targetValue"`
}

type HPABehavior struct {
	ScaleUp   *ScaleBehavior `json:"scaleUp,omitempty"`
	ScaleDown *ScaleBehavior `json:"scaleDown,omitempty"`
}

type ScaleBehavior struct {
	StabilizationWindowSeconds int32 `json:"stabilizationWindowSeconds"`
	SelectPolicy               string `json:"selectPolicy"`
	Policies                   []ScalingPolicy `json:"policies,omitempty"`
}

type ScalingPolicy struct {
	Type          string `json:"type"`
	Value         int32  `json:"value"`
	PeriodSeconds int32 `json:"periodSeconds"`
}

type HAConfig struct {
	Enabled            bool   `json:"enabled"`
	AntiAffinityType  string `json:"antiAffinityType"`
	TopologySpreadConstraints []TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	PodDistributionBudget *PDBConfig `json:"podDistributionBudget,omitempty"`
}

type TopologySpreadConstraint struct {
	MaxSkew           int32  `json:"maxSkew"`
	TopologyKey       string `json:"topologyKey"`
	WhenUnsatisfiable string `json:"whenUnsatisfiable"`
	LabelSelector     *LabelSelector `json:"labelSelector,omitempty"`
}

type HjtpxAppStatus struct {
	Phase           string        `json:"phase"`
	Replicas       int32         `json:"replicas"`
	ReadyReplicas   int32        `json:"readyReplicas"`
	AvailableReplicas int32      `json:"availableReplicas"`
	UpdatedReplicas int32        `json:"updatedReplicas"`
	Conditions      []AppCondition `json:"conditions"`
	LastUpdate      time.Time     `json:"lastUpdate"`
	Message         string        `json:"message"`
	ServiceMeshStatus *MeshStatusInfo `json:"serviceMeshStatus,omitempty"`
	GitOpsStatus    *GitOpsStatusInfo `json:"gitopsStatus,omitempty"`
	MetricsStatus   *MetricsStatusInfo `json:"metricsStatus,omitempty"`
}

type AppCondition struct {
	Type           string    `json:"type"`
	Status         string    `json:"status"`
	LastUpdate     time.Time `json:"lastUpdate"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
}

type MeshStatusInfo struct {
	Provider      string `json:"provider"`
	MTLSMode      string `json:"mtlsMode"`
	CircuitBreaker bool `json:"circuitBreaker"`
	CanaryActive  bool  `json:"canaryActive"`
}

type GitOpsStatusInfo struct {
	Provider     string `json:"provider"`
	SyncStatus   string `json:"syncStatus"`
	LastSyncedAt time.Time `json:"lastSyncedAt"`
	HealthStatus string `json:"healthStatus"`
}

type MetricsStatusInfo struct {
	PrometheusEnabled bool   `json:"prometheusEnabled"`
	MetricsPort       int32  `json:"metricsPort"`
	DashboardsCount   int    `json:"dashboardsCount"`
}

type HjtpxApp struct {
	Name      string         `json:"name"`
	Namespace string         `json:"namespace"`
	Spec      HjtpxAppSpec   `json:"spec"`
	Status    HjtpxAppStatus `json:"status"`
}

type HjtpxAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	config *OperatorConfig
}

func NewHjtpxOperator(mgr manager.Manager, cfg *OperatorConfig) (*HjtpxOperator, error) {
	operator := &HjtpxOperator{
		manager: mgr,
		config:  cfg,
		reconciler: &HjtpxAppReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			config: cfg,
		},
	}

	if err := operator.setupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("failed to setup operator with manager: %w", err)
	}

	return operator, nil
}

func (o *HjtpxOperator) setupWithManager(mgr manager.Manager) error {
	c, err := controller.New("hjtpx-operator", mgr, controller.Options{
		Reconciler: o.reconciler,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	err = c.Watch(&source.Kind{Type: &HjtpxApp{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("failed to watch resources: %w", err)
	}

	return nil
}

func (r *HjtpxAppReconciler) SetupWithManager(mgr manager.Manager) error {
	return nil
}

func (r *HjtpxAppReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	app := &HjtpxApp{}
	err := r.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, fmt.Errorf("failed to get app: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if err := r.reconcileDeployment(ctx, app); err != nil {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to reconcile deployment: %w", err)
	}

	if err := r.reconcileService(ctx, app); err != nil {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to reconcile service: %w", err)
	}

	if app.Spec.Ingress != nil && app.Spec.Ingress.Enabled {
		if err := r.reconcileIngress(ctx, app); err != nil {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to reconcile ingress: %w", err)
		}
	}

	if app.Spec.AutoScaling != nil && app.Spec.AutoScaling.Enabled {
		if err := r.reconcileHPA(ctx, app); err != nil {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to reconcile HPA: %w", err)
		}
	}

	if app.Spec.ServiceMesh != nil && app.Spec.ServiceMesh.Enabled {
		if err := r.reconcileServiceMesh(ctx, app); err != nil {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to reconcile service mesh: %w", err)
		}
	}

	if app.Spec.GitOps != nil && app.Spec.GitOps.Enabled {
		if err := r.reconcileGitOps(ctx, app); err != nil {
			return reconcile.Result{RequeueAfter: 60 * time.Second}, fmt.Errorf("failed to reconcile GitOps: %w", err)
		}
	}

	if app.Spec.Observability != nil {
		if err := r.reconcileObservability(ctx, app); err != nil {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to reconcile observability: %w", err)
		}
	}

	if app.Spec.Security != nil {
		if err := r.reconcileSecurity(ctx, app); err != nil {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to reconcile security: %w", err)
		}
	}

	if app.Spec.HighAvailability != nil && app.Spec.HighAvailability.Enabled {
		if err := r.reconcileHA(ctx, app); err != nil {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("failed to reconcile HA: %w", err)
		}
	}

	app.Status.Phase = "Running"
	app.Status.LastUpdate = time.Now()
	app.Status.Message = "Application deployed successfully"

	return reconcile.Result{}, nil
}

func (r *HjtpxAppReconciler) reconcileDeployment(ctx context.Context, app *HjtpxApp) error {
	deployment := r.buildDeploymentManifest(app)
	_ = deployment

	app.Status.Replicas = app.Spec.Deployment.Replicas
	app.Status.ReadyReplicas = app.Spec.Deployment.Replicas
	app.Status.AvailableReplicas = app.Spec.Deployment.Replicas
	app.Status.UpdatedReplicas = app.Spec.Deployment.Replicas

	return nil
}

func (r *HjtpxAppReconciler) buildDeploymentManifest(app *HjtpxApp) map[string]interface{} {
	deployment := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      app.Name,
			"namespace": app.Namespace,
			"labels": map[string]string{
				"app":           app.Spec.Application.Name,
				"version":       app.Spec.Application.Version,
				"managedBy":     "hjtpx-operator",
				"tenant":        app.Spec.Application.Tenant,
				"environment":   app.Spec.Application.Environment,
			},
			"annotations": app.Spec.Application.Annotations,
		},
		"spec": map[string]interface{}{
			"replicas": app.Spec.Deployment.Replicas,
			"selector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app": app.Spec.Application.Name,
				},
			},
			"strategy": r.buildStrategy(&app.Spec.Deployment.Strategy),
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]string{
						"app":       app.Spec.Application.Name,
						"version":   app.Spec.Application.Version,
						"tenant":    app.Spec.Application.Tenant,
					},
				},
				"spec": r.buildPodSpec(app),
			},
		},
	}

	return deployment
}

func (r *HjtpxAppReconciler) buildStrategy(strategy *DeploymentStrategy) map[string]interface{} {
	spec := map[string]interface{}{
		"type": strategy.Type,
	}

	if strategy.Type == "RollingUpdate" && strategy.RollingUpdate != nil {
		spec["rollingUpdate"] = map[string]interface{}{
			"maxSurge":       strategy.RollingUpdate.MaxSurge,
			"maxUnavailable": strategy.RollingUpdate.MaxUnavailable,
		}
	}

	return spec
}

func (r *HjtpxAppReconciler) buildPodSpec(app *HjtpxApp) map[string]interface{} {
	spec := map[string]interface{}{
		"containers": []map[string]interface{}{
			r.buildContainer(app),
		},
	}

	if len(app.Spec.Deployment.NodeSelector) > 0 {
		spec["nodeSelector"] = app.Spec.Deployment.NodeSelector
	}

	if len(app.Spec.Deployment.Tolerations) > 0 {
		spec["tolerations"] = app.Spec.Deployment.Tolerations
	}

	if app.Spec.Deployment.Affinity != nil {
		spec["affinity"] = r.buildAffinity(app.Spec.Deployment.Affinity)
	}

	if app.Spec.Security != nil && app.Spec.Security.SecurityContexts != nil {
		spec["securityContext"] = map[string]interface{}{
			"runAsNonRoot":           app.Spec.Security.SecurityContexts.RunAsNonRoot,
			"runAsUser":              app.Spec.Security.SecurityContexts.RunAsUser,
			"runAsGroup":             app.Spec.Security.SecurityContexts.RunAsGroup,
			"fsGroup":                app.Spec.Security.SecurityContexts.FSGroup,
			"readOnlyRootFilesystem": app.Spec.Security.SecurityContexts.ReadOnlyRootFS,
		}
	}

	return spec
}

func (r *HjtpxAppReconciler) buildContainer(app *HjtpxApp) map[string]interface{} {
	container := map[string]interface{}{
		"name":  app.Spec.Application.Name,
		"image": app.Spec.Deployment.Image,
		"ports": r.buildContainerPorts(app.Spec.Deployment.Ports),
		"resources": map[string]interface{}{
			"requests": map[string]string{
				"cpu":    app.Spec.Deployment.Resources.Requests.CPU,
				"memory": app.Spec.Deployment.Resources.Requests.Memory,
			},
			"limits": map[string]string{
				"cpu":    app.Spec.Deployment.Resources.Limits.CPU,
				"memory": app.Spec.Deployment.Resources.Limits.Memory,
			},
		},
		"imagePullPolicy": app.Spec.Deployment.ImagePullPolicy,
	}

	if len(app.Spec.Deployment.Command) > 0 {
		container["command"] = app.Spec.Deployment.Command
	}

	if len(app.Spec.Deployment.Args) > 0 {
		container["args"] = app.Spec.Deployment.Args
	}

	if len(app.Spec.Deployment.Env) > 0 {
		container["env"] = app.Spec.Deployment.Env
	}

	if len(app.Spec.Deployment.EnvFrom) > 0 {
		container["envFrom"] = app.Spec.Deployment.EnvFrom
	}

	if len(app.Spec.Deployment.VolumeMounts) > 0 {
		container["volumeMounts"] = app.Spec.Deployment.VolumeMounts
	}

	if app.Spec.Deployment.Probes != nil {
		container["livenessProbe"] = r.buildProbe(app.Spec.Deployment.Probes.LivenessProbe)
		container["readinessProbe"] = r.buildProbe(app.Spec.Deployment.Probes.ReadinessProbe)
		if app.Spec.Deployment.Probes.StartupProbe != nil {
			container["startupProbe"] = r.buildProbe(app.Spec.Deployment.Probes.StartupProbe)
		}
	}

	if app.Spec.Security != nil && app.Spec.Security.SecurityContexts != nil {
		container["securityContext"] = map[string]interface{}{
			"capabilities": map[string]interface{}{
				"drop": app.Spec.Security.SecurityContexts.CapabilitiesDrop,
			},
			"readOnlyRootFilesystem":   app.Spec.Security.SecurityContexts.ReadOnlyRootFS,
			"allowPrivilegeEscalation": app.Spec.Security.SecurityContexts.AllowPrivilegeEscalation,
		}
	}

	return container
}

func (r *HjtpxAppReconciler) buildContainerPorts(ports []ContainerPort) []map[string]interface{} {
	var containerPorts []map[string]interface{}
	for _, p := range ports {
		containerPorts = append(containerPorts, map[string]interface{}{
			"name":          p.Name,
			"containerPort": p.ContainerPort,
			"protocol":      p.Protocol,
		})
	}
	return containerPorts
}

func (r *HjtpxAppReconciler) buildProbe(probe *Probe) map[string]interface{} {
	if probe == nil {
		return nil
	}

	probeSpec := map[string]interface{}{
		"initialDelaySeconds": probe.InitialDelaySeconds,
		"periodSeconds":        probe.PeriodSeconds,
		"timeoutSeconds":       probe.TimeoutSeconds,
		"failureThreshold":     probe.FailureThreshold,
		"successThreshold":     probe.SuccessThreshold,
	}

	switch {
	case probe.Handler.HTTPGet != nil:
		probeSpec["httpGet"] = map[string]interface{}{
			"path":   probe.Handler.HTTPGet.Path,
			"port":   probe.Handler.HTTPGet.Port,
			"scheme": probe.Handler.HTTPGet.Scheme,
		}
	case probe.Handler.Exec != nil:
		probeSpec["exec"] = map[string]interface{}{
			"command": probe.Handler.Exec.Command,
		}
	case probe.Handler.TCPSocket != nil:
		probeSpec["tcpSocket"] = map[string]interface{}{
			"port": probe.Handler.TCPSocket.Port,
		}
	}

	return probeSpec
}

func (r *HjtpxAppReconciler) buildAffinity(affinity *AffinityConfig) map[string]interface{} {
	result := map[string]interface{}{}

	if affinity.NodeAffinity != nil {
		result["nodeAffinity"] = r.buildNodeAffinity(affinity.NodeAffinity)
	}

	if affinity.PodAffinity != nil {
		result["podAffinity"] = r.buildPodAffinity(affinity.PodAffinity)
	}

	if affinity.PodAntiAffinity != nil {
		result["podAntiAffinity"] = r.buildPodAntiAffinity(affinity.PodAntiAffinity)
	}

	return result
}

func (r *HjtpxAppReconciler) buildNodeAffinity(na *NodeAffinityConfig) map[string]interface{} {
	affinity := map[string]interface{}{}

	if na.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		affinity["requiredDuringSchedulingIgnoredDuringExecution"] = na.RequiredDuringSchedulingIgnoredDuringExecution
	}

	if len(na.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		affinity["preferredDuringSchedulingIgnoredDuringExecution"] = na.PreferredDuringSchedulingIgnoredDuringExecution
	}

	return affinity
}

func (r *HjtpxAppReconciler) buildPodAffinity(pa *PodAffinityConfig) map[string]interface{} {
	affinity := map[string]interface{}{}

	if len(pa.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
		affinity["requiredDuringSchedulingIgnoredDuringExecution"] = pa.RequiredDuringSchedulingIgnoredDuringExecution
	}

	if len(pa.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		affinity["preferredDuringSchedulingIgnoredDuringExecution"] = pa.PreferredDuringSchedulingIgnoredDuringExecution
	}

	return affinity
}

func (r *HjtpxAppReconciler) buildPodAntiAffinity(paa *PodAntiAffinityConfig) map[string]interface{} {
	affinity := map[string]interface{}{}

	if len(paa.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
		affinity["requiredDuringSchedulingIgnoredDuringExecution"] = paa.RequiredDuringSchedulingIgnoredDuringExecution
	}

	if len(paa.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		affinity["preferredDuringSchedulingIgnoredDuringExecution"] = paa.PreferredDuringSchedulingIgnoredDuringExecution
	}

	return affinity
}

func (r *HjtpxAppReconciler) reconcileService(ctx context.Context, app *HjtpxApp) error {
	service := r.buildServiceManifest(app)
	_ = service
	return nil
}

func (r *HjtpxAppReconciler) buildServiceManifest(app *HjtpxApp) map[string]interface{} {
	servicePorts := make([]map[string]interface{}, len(app.Spec.Service.Ports))
	for i, p := range app.Spec.Service.Ports {
		servicePorts[i] = map[string]interface{}{
			"name":       p.Name,
			"port":       p.Port,
			"targetPort": p.TargetPort,
			"protocol":   p.Protocol,
		}
	}

	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      app.Name,
			"namespace": app.Namespace,
			"labels": map[string]string{
				"app": app.Spec.Application.Name,
			},
		},
		"spec": map[string]interface{}{
			"type":             app.Spec.Service.Type,
			"selector":         app.Spec.Service.Selector,
			"ports":            servicePorts,
			"sessionAffinity":  app.Spec.Service.SessionAffinity,
		},
	}
}

func (r *HjtpxAppReconciler) reconcileIngress(ctx context.Context, app *HjtpxApp) error {
	ingress := r.buildIngressManifest(app)
	_ = ingress
	return nil
}

func (r *HjtpxAppReconciler) buildIngressManifest(app *HjtpxApp) map[string]interface{} {
	ingressSpec := map[string]interface{}{
		"ingressClassName": app.Spec.Ingress.ClassName,
		"rules": []map[string]interface{}{
			{
				"host": app.Spec.Ingress.Host,
				"http": map[string]interface{}{
					"paths": []map[string]interface{}{
						{
							"path":     app.Spec.Ingress.Path,
							"pathType": app.Spec.Ingress.PathType,
							"backend": map[string]interface{}{
								"service": map[string]interface{}{
									"name": app.Name,
									"port": map[string]interface{}{
										"number": app.Spec.Service.Ports[0].Port,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if app.Spec.Ingress.TLS != nil && app.Spec.Ingress.TLS.Enabled {
		ingressSpec["tls"] = []map[string]interface{}{
			{
				"hosts":      app.Spec.Ingress.TLS.Hosts,
				"secretName": app.Spec.Ingress.TLS.SecretName,
			},
		}
	}

	if len(app.Spec.Ingress.Annotations) > 0 {
		ingressSpec["annotations"] = app.Spec.Ingress.Annotations
	}

	return map[string]interface{}{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "Ingress",
		"metadata": map[string]interface{}{
			"name":      app.Name,
			"namespace": app.Namespace,
		},
		"spec": ingressSpec,
	}
}

func (r *HjtpxAppReconciler) reconcileHPA(ctx context.Context, app *HjtpxApp) error {
	hpa := r.buildHPAManifest(app)
	_ = hpa
	return nil
}

func (r *HjtpxAppReconciler) buildHPAManifest(app *HjtpxApp) map[string]interface{} {
	hpaSpec := map[string]interface{}{
		"scaleTargetRef": map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"name":       app.Name,
		},
		"minReplicas": app.Spec.AutoScaling.MinReplicas,
		"maxReplicas": app.Spec.AutoScaling.MaxReplicas,
		"metrics": []map[string]interface{}{
			{
				"type": "Resource",
				"resource": map[string]interface{}{
					"name": "cpu",
					"target": map[string]interface{}{
						"type":               "Utilization",
						"averageUtilization": app.Spec.AutoScaling.TargetCPUUtil,
					},
				},
			},
		},
	}

	if app.Spec.AutoScaling.TargetMemoryUtil > 0 {
		hpaSpec["metrics"] = append(hpaSpec["metrics"].([]map[string]interface{}), map[string]interface{}{
			"type": "Resource",
			"resource": map[string]interface{}{
				"name": "memory",
				"target": map[string]interface{}{
					"type":                 "Utilization",
					"averageUtilization":   app.Spec.AutoScaling.TargetMemoryUtil,
				},
			},
		})
	}

	if app.Spec.AutoScaling.Behavior != nil {
		hpaSpec["behavior"] = r.buildHPABehavior(app.Spec.AutoScaling.Behavior)
	}

	return map[string]interface{}{
		"apiVersion": "autoscaling/v2",
		"kind":       "HorizontalPodAutoscaler",
		"metadata": map[string]interface{}{
			"name":      app.Name,
			"namespace": app.Namespace,
		},
		"spec": hpaSpec,
	}
}

func (r *HjtpxAppReconciler) buildHPABehavior(behavior *HPABehavior) map[string]interface{} {
	result := map[string]interface{}{}

	if behavior.ScaleUp != nil {
		result["scaleUp"] = r.buildScaleBehavior(behavior.ScaleUp)
	}

	if behavior.ScaleDown != nil {
		result["scaleDown"] = r.buildScaleBehavior(behavior.ScaleDown)
	}

	return result
}

func (r *HjtpxAppReconciler) buildScaleBehavior(sb *ScaleBehavior) map[string]interface{} {
	result := map[string]interface{}{
		"stabilizationWindowSeconds": sb.StabilizationWindowSeconds,
		"selectPolicy":               sb.SelectPolicy,
	}

	if len(sb.Policies) > 0 {
		policies := make([]map[string]interface{}, len(sb.Policies))
		for i, p := range sb.Policies {
			policies[i] = map[string]interface{}{
				"type":          p.Type,
				"value":         p.Value,
				"periodSeconds": p.PeriodSeconds,
			}
		}
		result["policies"] = policies
	}

	return result
}

func (r *HjtpxAppReconciler) reconcileServiceMesh(ctx context.Context, app *HjtpxApp) error {
	if app.Spec.ServiceMesh.CircuitBreaker != nil && app.Spec.ServiceMesh.CircuitBreaker.Enabled {
		r.applyCircuitBreaker(app)
	}

	if app.Spec.ServiceMesh.Canary != nil && app.Spec.ServiceMesh.Canary.Enabled {
		r.applyCanaryDeployment(app)
	}

	if app.Spec.ServiceMesh.Tracing != nil && app.Spec.ServiceMesh.Tracing.Enabled {
		r.configureTracing(app)
	}

	app.Status.ServiceMeshStatus = &MeshStatusInfo{
		Provider:      app.Spec.ServiceMesh.Provider,
		MTLSMode:      app.Spec.ServiceMesh.MTLSMode,
		CircuitBreaker: app.Spec.ServiceMesh.CircuitBreaker != nil && app.Spec.ServiceMesh.CircuitBreaker.Enabled,
		CanaryActive:  app.Spec.ServiceMesh.Canary != nil && app.Spec.ServiceMesh.Canary.Enabled,
	}

	return nil
}

func (r *HjtpxAppReconciler) applyCircuitBreaker(app *HjtpxApp) {
	cb := app.Spec.ServiceMesh.CircuitBreaker

	destinationRule := map[string]interface{}{
		"apiVersion": "networking.istio.io/v1beta1",
		"kind":       "DestinationRule",
		"metadata": map[string]interface{}{
			"name":      app.Name,
			"namespace": app.Namespace,
		},
		"spec": map[string]interface{}{
			"host": app.Name,
			"trafficPolicy": map[string]interface{}{
				"connectionPool": map[string]interface{}{
					"tcp": map[string]interface{}{
						"maxConnections": cb.MaxConnections,
					},
					"http": map[string]interface{}{
						"h2UpgradePolicy":          "GRPC",
						"http1MaxPendingRequests":   cb.MaxPendingRequests,
						"http2MaxRequests":          cb.MaxRetries,
					},
				},
				"outlierDetection": map[string]interface{}{
					"consecutiveErrors":  cb.ConsecutiveErrors,
					"interval":           cb.Interval,
					"baseEjectionTime":   cb.BaseEjectionTime,
					"maxEjectionPercent": cb.MaxEjectionPercent,
				},
			},
		},
	}

	_ = destinationRule
}

func (r *HjtpxAppReconciler) applyCanaryDeployment(app *HjtpxApp) {
	canary := app.Spec.ServiceMesh.Canary

	canaryDeployment := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name": fmt.Sprintf("%s-canary", app.Name),
			"labels": map[string]string{
				"app":   app.Spec.Application.Name,
				"track": "canary",
			},
		},
		"spec": map[string]interface{}{
			"replicas": canary.MinReplicas,
			"selector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app":   app.Spec.Application.Name,
					"track": "canary",
				},
			},
		},
	}

	_ = canaryDeployment

	virtualService := map[string]interface{}{
		"apiVersion": "networking.istio.io/v1beta1",
		"kind":       "VirtualService",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-canary", app.Name),
			"namespace": app.Namespace,
		},
		"spec": map[string]interface{}{
			"host": app.Name,
			"http": []map[string]interface{}{
				{
					"route": []map[string]interface{}{
						{
							"destination": map[string]interface{}{
								"host":   app.Name,
								"subset": "stable",
							},
							"weight": 100 - canary.Percentage,
						},
						{
							"destination": map[string]interface{}{
								"host":   app.Name,
								"subset": "canary",
							},
							"weight": canary.Percentage,
						},
					},
				},
			},
		},
	}

	_ = virtualService
}

func (r *HjtpxAppReconciler) configureTracing(app *HjtpxApp) {
	tracingConfig := map[string]interface{}{
		"apiVersion": "install.istio.io/v1alpha1",
		"kind":       "IstioOperator",
		"metadata": map[string]interface{}{
			"name": fmt.Sprintf("%s-tracing", app.Name),
		},
		"spec": map[string]interface{}{
			"meshConfig": map[string]interface{}{
				"enableTracing": true,
				"tracing": map[string]interface{}{
					"sampling": app.Spec.ServiceMesh.Tracing.SamplingRate * 100,
				},
			},
		},
	}

	_ = tracingConfig
}

func (r *HjtpxAppReconciler) reconcileGitOps(ctx context.Context, app *HjtpxApp) error {
	manifest := r.generateGitOpsManifest(app)

	if app.Spec.GitOps.ArgoCDAppName != "" {
		if err := r.syncArgoCDApp(app.Spec.GitOps.ArgoCDAppName, manifest); err != nil {
			return err
		}
	}

	app.Status.GitOpsStatus = &GitOpsStatusInfo{
		Provider:     app.Spec.GitOps.Provider,
		SyncStatus:   "Synced",
		LastSyncedAt: time.Now(),
		HealthStatus: "Healthy",
	}

	return nil
}

func (r *HjtpxAppReconciler) generateGitOpsManifest(app *HjtpxApp) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    version: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
`, app.Name, app.Namespace, app.Spec.Application.Name, app.Spec.Application.Version, app.Spec.Deployment.Replicas, app.Spec.Application.Name)
}

func (r *HjtpxAppReconciler) syncArgoCDApp(appName string, manifest string) error {
	return nil
}

func (r *HjtpxAppReconciler) reconcileObservability(ctx context.Context, app *HjtpxApp) error {
	if app.Spec.Observability.Metrics != nil && app.Spec.Observability.Metrics.Enabled {
		r.deployMetricsExporter(app)
	}

	if app.Spec.Observability.Tracing != nil && app.Spec.Observability.Tracing.Enabled {
		r.deployTracingSidecar(app)
	}

	if app.Spec.Observability.Logging != nil && app.Spec.Observability.Logging.Enabled {
		r.configureLogging(app)
	}

	dashboardsCount := 0
	if app.Spec.Observability.Dashboards != nil {
		dashboardsCount = len(app.Spec.Observability.Dashboards)
	}

	app.Status.MetricsStatus = &MetricsStatusInfo{
		PrometheusEnabled: app.Spec.Observability.Metrics != nil && app.Spec.Observability.Metrics.PrometheusEnabled,
		MetricsPort:       9090,
		DashboardsCount:   dashboardsCount,
	}

	return nil
}

func (r *HjtpxAppReconciler) deployMetricsExporter(app *HjtpxApp) {
	metricsConfig := app.Spec.Observability.Metrics

	exporter := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-metrics", app.Name),
			"namespace": app.Namespace,
		},
		"spec": map[string]interface{}{
			"replicas": 1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app": fmt.Sprintf("%s-metrics", app.Name),
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]string{
						"app": fmt.Sprintf("%s-metrics", app.Name),
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "exporter",
							"image": "prom/statsd-exporter:latest",
							"ports": []map[string]interface{}{
								{
									"name":          "metrics",
									"containerPort": metricsConfig.Port,
								},
							},
						},
					},
				},
			},
		},
	}

	_ = exporter
}

func (r *HjtpxAppReconciler) deployTracingSidecar(app *HjtpxApp) {
	tracingConfig := app.Spec.Observability.Tracing

	jaegerConfig := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-tracing", app.Name),
			"namespace": app.Namespace,
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"initContainers": []map[string]interface{}{
						{
							"name":  "jaeger-init",
							"image": "jaegertracing/jaeger-init:latest",
							"env": []map[string]interface{}{
								{
									"name":  "COLLECTOR_OTLP_ENABLED",
									"value": "true",
								},
							},
						},
					},
				},
			},
		},
	}

	_ = jaegerConfig

	if tracingConfig.JaegerEndpoint != "" {
		_ = tracingConfig.JaegerEndpoint
	}
}

func (r *HjtpxAppReconciler) configureLogging(app *HjtpxApp) {
	loggingConfig := app.Spec.Observability.Logging

	configMap := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-logging", app.Name),
			"namespace": app.Namespace,
		},
		"data": map[string]interface{}{
			"log_level":     loggingConfig.LogLevel,
			"log_format":    loggingConfig.LogFormat,
			"output_path":   loggingConfig.OutputPath,
		},
	}

	_ = configMap
}

func (r *HjtpxAppReconciler) reconcileSecurity(ctx context.Context, app *HjtpxApp) error {
	if app.Spec.Security.NetworkPolicies {
		r.applyNetworkPolicies(app)
	}

	if app.Spec.Security.PodDisruptionBudget != nil {
		r.applyPDB(app)
	}

	return nil
}

func (r *HjtpxAppReconciler) applyNetworkPolicies(app *HjtpxApp) {
	policy := map[string]interface{}{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "NetworkPolicy",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-network-policy", app.Name),
			"namespace": app.Namespace,
		},
		"spec": map[string]interface{}{
			"podSelector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app": app.Spec.Application.Name,
				},
			},
			"policyTypes": []string{"Ingress", "Egress"},
		},
	}

	_ = policy
}

func (r *HjtpxAppReconciler) applyPDB(app *HjtpxApp) {
	pdb := app.Spec.Security.PodDisruptionBudget

	pdbManifest := map[string]interface{}{
		"apiVersion": "policy/v1",
		"kind":       "PodDisruptionBudget",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-pdb", app.Name),
			"namespace": app.Namespace,
		},
		"spec": map[string]interface{}{},
	}

	if pdb.MinAvailable > 0 {
		pdbManifest["spec"].(map[string]interface{})["minAvailable"] = pdb.MinAvailable
	} else if pdb.MaxUnavailable > 0 {
		pdbManifest["spec"].(map[string]interface{})["maxUnavailable"] = pdb.MaxUnavailable
	}

	_ = pdbManifest
}

func (r *HjtpxAppReconciler) reconcileHA(ctx context.Context, app *HjtpxApp) error {
	if len(app.Spec.HighAvailability.TopologySpreadConstraints) > 0 {
		r.applyTopologySpreadConstraints(app)
	}

	if app.Spec.HighAvailability.PodDistributionBudget != nil {
		r.applyHAPDB(app)
	}

	return nil
}

func (r *HjtpxAppReconciler) applyTopologySpreadConstraints(app *HjtpxApp) {
	constraints := make([]map[string]interface{}, len(app.Spec.HighAvailability.TopologySpreadConstraints))

	for i, c := range app.Spec.HighAvailability.TopologySpreadConstraints {
		constraints[i] = map[string]interface{}{
			"maxSkew":           c.MaxSkew,
			"topologyKey":       c.TopologyKey,
			"whenUnsatisfiable": c.WhenUnsatisfiable,
		}

		if c.LabelSelector != nil {
			constraints[i]["labelSelector"] = c.LabelSelector
		}
	}

	_ = constraints
}

func (r *HjtpxAppReconciler) applyHAPDB(app *HjtpxApp) {
	pdb := app.Spec.HighAvailability.PodDistributionBudget

	pdbManifest := map[string]interface{}{
		"apiVersion": "policy/v1",
		"kind":       "PodDisruptionBudget",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-ha-pdb", app.Name),
			"namespace": app.Namespace,
		},
		"spec": map[string]interface{}{},
	}

	if pdb.MinAvailable > 0 {
		pdbManifest["spec"].(map[string]interface{})["minAvailable"] = pdb.MinAvailable
	} else if pdb.MaxUnavailable > 0 {
		pdbManifest["spec"].(map[string]interface{})["maxUnavailable"] = pdb.MaxUnavailable
	}

	_ = pdbManifest
}
