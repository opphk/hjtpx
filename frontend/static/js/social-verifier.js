class SocialVerifier {
    constructor(config = {}) {
        this.config = {
            apiEndpoint: '/api/v1/social',
            avatarRequired: true,
            enableFriendVerification: true,
            enableCommunityTrust: true,
            crossVerificationRequired: true,
            minConnections: 3,
            minTrustScore: 0.6,
            ...config
        };

        this.session = null;
        this.avatar = null;
        this.socialGraph = null;
        this.communities = [];
        this.friendRequest = null;
        this.initialized = false;
    }

    async init(userId) {
        if (this.initialized) {
            this.destroy();
        }

        this.userId = userId;
        this.initialized = true;

        console.log('Social Verifier initialized for user:', userId);
    }

    async createAvatar(style = 'realistic', name = '') {
        try {
            const response = await fetch(`${this.config.apiEndpoint}/avatar`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    user_id: this.userId,
                    style: style,
                    name: name
                })
            });

            if (!response.ok) {
                throw new Error('Failed to create avatar');
            }

            const data = await response.json();
            this.avatar = data.avatar;
            return this.avatar;
        } catch (error) {
            console.error('Failed to create avatar:', error);
            return this.createLocalAvatar(style, name);
        }
    }

    createLocalAvatar(style, name) {
        this.avatar = {
            id: `avatar_${this.userId}_${Date.now()}`,
            user_id: this.userId,
            style: style,
            name: name || `User_${this.userId.substring(0, 8)}`,
            avatar_url: `/avatars/${style}/${this.userId}.png`,
            emotion: 'neutral',
            expression: {
                eyes: 'open',
                mouth: 'neutral',
                emotion: 'neutral',
                intensity: 0.5
            },
            position: [0, 0, 0],
            rotation: [0, 0, 0],
            scale: 1.0,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString()
        };

        return this.avatar;
    }

    async updateAvatarExpression(emotion, intensity = 0.8) {
        if (!this.avatar) {
            console.warn('No avatar to update');
            return null;
        }

        this.avatar.emotion = emotion;
        this.avatar.expression = this.getExpressionFromEmotion(emotion, intensity);
        this.avatar.updated_at = new Date().toISOString();

        try {
            await fetch(`${this.config.apiEndpoint}/avatar/${this.avatar.id}/expression`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    emotion: emotion,
                    intensity: intensity
                })
            });
        } catch (error) {
            console.warn('Failed to sync avatar expression:', error);
        }

        this.renderAvatar();
        return this.avatar;
    }

    getExpressionFromEmotion(emotion, intensity) {
        const expressions = {
            happy: { eyes: 'curved', mouth: 'smile', emotion: 'happy', intensity: intensity },
            sad: { eyes: 'down', mouth: 'frown', emotion: 'sad', intensity: intensity },
            angry: { eyes: 'narrowed', mouth: 'grimace', emotion: 'angry', intensity: intensity },
            surprised: { eyes: 'wide', mouth: 'open', emotion: 'surprised', intensity: intensity },
            neutral: { eyes: 'open', mouth: 'neutral', emotion: 'neutral', intensity: intensity },
            confused: { eyes: 'asymmetric', mouth: 'wavy', emotion: 'confused', intensity: intensity }
        };

        return expressions[emotion] || expressions.neutral;
    }

    async updateAvatarPose(position, rotation, scale = 1.0) {
        if (!this.avatar) {
            console.warn('No avatar to update');
            return null;
        }

        this.avatar.position = position;
        this.avatar.rotation = rotation;
        this.avatar.scale = scale;
        this.avatar.updated_at = new Date().toISOString();

        try {
            await fetch(`${this.config.apiEndpoint}/avatar/${this.avatar.id}/pose`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    position: position,
                    rotation: rotation,
                    scale: scale
                })
            });
        } catch (error) {
            console.warn('Failed to sync avatar pose:', error);
        }

        this.renderAvatar();
        return this.avatar;
    }

    renderAvatar() {
        let container = document.querySelector('.avatar-container');
        if (!container) {
            container = document.createElement('div');
            container.className = 'avatar-container';
            document.body.appendChild(container);
        }

        container.innerHTML = `
            <div class="avatar-wrapper">
                <div class="avatar-avatar avatar-${this.avatar.style}">
                    <div class="avatar-face">
                        <div class="avatar-eyes avatar-eyes-${this.avatar.expression.eyes}"></div>
                        <div class="avatar-mouth avatar-mouth-${this.avatar.expression.mouth}"></div>
                    </div>
                </div>
                <div class="avatar-info">
                    <span class="avatar-name">${this.avatar.name}</span>
                    <span class="avatar-emotion">${this.avatar.emotion}</span>
                </div>
            </div>
        `;

        container.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            z-index: 1000;
        `;
    }

    async buildSocialGraph(connectionIds) {
        if (!connectionIds || connectionIds.length < this.config.minConnections) {
            console.warn(`Need at least ${this.config.minConnections} connections`);
        }

        try {
            const response = await fetch(`${this.config.apiEndpoint}/graph`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    user_id: this.userId,
                    connection_ids: connectionIds
                })
            });

            if (!response.ok) {
                throw new Error('Failed to build social graph');
            }

            const data = await response.json();
            this.socialGraph = data.graph;
            return this.socialGraph;
        } catch (error) {
            console.error('Failed to build social graph:', error);
            return this.buildLocalSocialGraph(connectionIds);
        }
    }

    buildLocalSocialGraph(connectionIds) {
        const connections = connectionIds.map((friendId, index) => ({
            id: `conn_${index}`,
            user_id: this.userId,
            friend_id: friendId,
            connection_type: 'friend',
            strength: 0.5 + Math.random() * 0.4,
            verified: Math.random() > 0.5,
            created_at: new Date().toISOString()
        }));

        const verifiedCount = connections.filter(c => c.verified).length;
        const avgStrength = connections.reduce((sum, c) => sum + c.strength, 0) / connections.length;

        this.socialGraph = {
            user_id: this.userId,
            connections: connections,
            friends_count: connections.length,
            verified_friends: verifiedCount,
            trust_score: Math.min(1, avgStrength * (1 + verifiedCount * 0.1)),
            community_score: avgStrength,
            risk_level: avgStrength < 0.5 ? 'medium' : 'low'
        };

        return this.socialGraph;
    }

    async createCommunity(name) {
        try {
            const response = await fetch(`${this.config.apiEndpoint}/community`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    name: name
                })
            });

            if (!response.ok) {
                throw new Error('Failed to create community');
            }

            const data = await response.json();
            const community = data.community;
            this.communities.push(community);
            return community;
        } catch (error) {
            console.error('Failed to create community:', error);
            return this.createLocalCommunity(name);
        }
    }

    createLocalCommunity(name) {
        const community = {
            community_id: `community_${Date.now()}`,
            community_name: name,
            member_count: 1,
            trust_level: 0.8,
            avg_trust_score: 0.75,
            verified_members: 1,
            created_at: new Date().toISOString()
        };

        this.communities.push(community);
        return community;
    }

    async createFriendVerificationRequest(friendIds, requestType = 'support', message = '') {
        if (!this.config.enableFriendVerification) {
            console.warn('Friend verification is not enabled');
            return null;
        }

        try {
            const response = await fetch(`${this.config.apiEndpoint}/friend-verification`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    user_id: this.userId,
                    friend_ids: friendIds,
                    request_type: requestType,
                    message: message
                })
            });

            if (!response.ok) {
                throw new Error('Failed to create friend verification request');
            }

            const data = await response.json();
            this.friendRequest = data.request;
            return this.friendRequest;
        } catch (error) {
            console.error('Failed to create friend verification request:', error);
            return this.createLocalFriendRequest(friendIds, requestType, message);
        }
    }

    createLocalFriendRequest(friendIds, requestType, message) {
        this.friendRequest = {
            id: `freq_${Date.now()}`,
            user_id: this.userId,
            friend_ids: friendIds,
            request_type: requestType,
            message: message,
            status: 'pending',
            created_at: new Date().toISOString(),
            responded_at: null,
            response: null
        };

        return this.friendRequest;
    }

    async respondToFriendRequest(requestId, friendId, accept, responseText = '') {
        try {
            const response = await fetch(`${this.config.apiEndpoint}/friend-verification/${requestId}/respond`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    friend_id: friendId,
                    accept: accept,
                    response: responseText
                })
            });

            if (!response.ok) {
                throw new Error('Failed to respond to friend request');
            }

            if (accept) {
                this.updateConnectionVerification(friendId);
            }

            return true;
        } catch (error) {
            console.error('Failed to respond to friend request:', error);
            return false;
        }
    }

    updateConnectionVerification(friendId) {
        if (!this.socialGraph) return;

        const connection = this.socialGraph.connections.find(c => c.friend_id === friendId);
        if (connection) {
            connection.verified = true;
            this.socialGraph.verified_friends++;
        }
    }

    async verify() {
        if (this.config.avatarRequired && !this.avatar) {
            return {
                is_valid: false,
                details: 'Avatar is required'
            };
        }

        const verificationData = {
            session_id: this.session?.id || `session_${Date.now()}`,
            user_id: this.userId,
            avatar_data: this.avatar,
            social_graph: this.socialGraph,
            communities: this.communities,
            friend_request: this.friendRequest,
            timestamp: Date.now()
        };

        try {
            const response = await fetch(`${this.config.apiEndpoint}/verify`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(verificationData)
            });

            if (!response.ok) {
                throw new Error('Verification request failed');
            }

            const result = await response.json();
            return result;
        } catch (error) {
            console.error('Verification failed:', error);
            return this.performLocalVerification();
        }
    }

    performLocalVerification() {
        const avatarScore = this.evaluateAvatar();
        const socialScore = this.evaluateSocialGraph();
        const trustScore = this.evaluateTrustScore();
        const friendSupportScore = this.evaluateFriendSupport();
        const communityBonus = this.calculateCommunityBonus();

        const confidence =
            avatarScore * 0.25 +
            socialScore * 0.30 +
            trustScore * 0.25 +
            friendSupportScore * 0.10 +
            communityBonus * 0.10;

        const overallScore = confidence;

        let riskLevel = 'low';
        if (overallScore < 0.4 || trustScore < 0.3) {
            riskLevel = 'high';
        } else if (overallScore < 0.6 || socialScore < 0.5) {
            riskLevel = 'medium';
        }

        return {
            is_valid: overallScore >= this.config.minTrustScore && socialScore >= 0.5 && riskLevel !== 'high',
            confidence: confidence,
            avatar_score: avatarScore,
            social_score: socialScore,
            trust_score: trustScore,
            friend_support_score: friendSupportScore,
            community_bonus: communityBonus,
            overall_score: overallScore,
            risk_level: riskLevel,
            details: `avatar: ${avatarScore.toFixed(2)}, social: ${socialScore.toFixed(2)}, trust: ${trustScore.toFixed(2)}, overall: ${overallScore.toFixed(2)}`
        };
    }

    evaluateAvatar() {
        if (!this.avatar) return 0;

        let score = 0;
        if (this.avatar.name) score += 0.2;
        if (this.avatar.avatar_url) score += 0.3;
        if (this.avatar.emotion) score += 0.2;
        if (this.avatar.expression?.intensity > 0) score += 0.1;
        if (this.avatar.position?.length >= 3) score += 0.2;

        return Math.min(1, score);
    }

    evaluateSocialGraph() {
        if (!this.socialGraph || this.socialGraph.connections.length === 0) {
            return 0;
        }

        const connectionScore = this.socialGraph.connections.length >= this.config.minConnections ? 0.5 : 0.3;
        const verifiedBonus = (this.socialGraph.verified_friends / Math.max(1, this.socialGraph.connections.length)) * 0.3;
        const communityBonus = this.socialGraph.community_score * 0.2;

        return Math.min(1, connectionScore + verifiedBonus + communityBonus);
    }

    evaluateTrustScore() {
        if (!this.socialGraph) return 0;

        let score = this.socialGraph.trust_score;
        const verifiedBonus = (this.socialGraph.verified_friends / Math.max(1, this.socialGraph.connections.length)) * 0.15;

        let communityBonus = 0;
        if (this.communities.length > 0) {
            communityBonus = this.communities.reduce((sum, c) => sum + c.trust_level * c.avg_trust_score, 0) / this.communities.length;
        }

        return Math.min(1, score + verifiedBonus + communityBonus * 0.2);
    }

    evaluateFriendSupport() {
        if (!this.friendRequest) return 0.5;

        const statusScore = this.friendRequest.status === 'accepted' ? 1.0 :
            this.friendRequest.status === 'pending' ? 0.6 : 0.2;

        const participationRatio = this.friendRequest.friend_ids.length / Math.max(1, this.friendRequest.friend_ids.length);

        return statusScore * 0.7 + participationRatio * 0.3;
    }

    calculateCommunityBonus() {
        if (this.communities.length === 0) return 0;

        const totalScore = this.communities.reduce((sum, c) => sum + c.trust_level * c.avg_trust_score, 0);
        const avgScore = totalScore / this.communities.length;
        const participationBonus = Math.min(0.2, this.communities.length * 0.05);

        return Math.min(1, avgScore + participationBonus);
    }

    getSocialGraph() {
        return this.socialGraph;
    }

    getCommunities() {
        return this.communities;
    }

    getFriendRequest() {
        return this.friendRequest;
    }

    reset() {
        this.avatar = null;
        this.socialGraph = null;
        this.communities = [];
        this.friendRequest = null;
    }

    destroy() {
        this.reset();
        this.initialized = false;
    }
}

window.SocialVerifier = SocialVerifier;
