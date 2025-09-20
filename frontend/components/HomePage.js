const HomePage = {
    props: {
        isConnected: {
            type: Boolean,
            default: false
        },
        connectionStatus: {
            type: String,
            default: 'Disconnected'
        }
    },
    
    emits: ['join-game', 'host-game'],
    
    template: `
        <div class="home-page">
            <!-- Header Section -->
            <div class="hero-section">
                <h1 class="main-title">QuizUS</h1>
                <p class="main-description">
                    Join multiplayer quiz competitions, test your knowledge, 
                    and compete with friends in real-time quiz battles!
                </p>
            </div>
            
            <!-- Statistics Section -->
            <div class="stats-section">
                <div class="stat-card">
                    <div class="stat-number">1,250</div>
                    <div class="stat-label">Sample questions</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">320</div>
                    <div class="stat-label">Manual quizzes</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">2,570</div>
                    <div class="stat-label">Total scores</div>
                </div>
            </div>
            
            <!-- Action Buttons Section -->
            <div class="action-section">
                <div class="action-buttons">
                    <button 
                        class="btn btn-primary"
                        @click="$emit('join-game')"
                        :disabled="!isConnected"
                    >
                        <span class="btn-icon">🎮</span>
                        Join Game
                    </button>
                    
                    <button 
                        class="btn btn-secondary"
                        @click="$emit('host-game')"
                        :disabled="!isConnected"
                    >
                        <span class="btn-icon">🎯</span>
                        Host Game
                    </button>
                </div>
                
                <!-- Connection Status -->
                <div class="connection-status" :class="connectionStatusClass">
                    <div class="status-indicator"></div>
                    <span class="status-text">{{ connectionStatus }}</span>
                </div>
            </div>
            
            <!-- Features Section -->
            <div class="features-section">
                <div class="feature-grid">
                    <div class="feature-card">
                        <div class="feature-icon">⚡</div>
                        <h3 class="feature-title">Real-time Competition</h3>
                        <p class="feature-description">
                            Compete with players in real-time with instant scoring and leaderboards
                        </p>
                    </div>
                    
                    <div class="feature-card">
                        <div class="feature-icon">🧠</div>
                        <h3 class="feature-title">Diverse Categories</h3>
                        <p class="feature-description">
                            Choose from thousands of questions across multiple categories and difficulty levels
                        </p>
                    </div>
                    
                    <div class="feature-card">
                        <div class="feature-icon">👥</div>
                        <h3 class="feature-title">Multiplayer Fun</h3>
                        <p class="feature-description">
                            Play with friends or join public rooms to meet new quiz enthusiasts
                        </p>
                    </div>
                </div>
            </div>
        </div>
    `,
    
    computed: {
        connectionStatusClass() {
            if (this.isConnected) return 'status-connected';
            if (this.connectionStatus === 'Connecting...' || this.connectionStatus.includes('Reconnecting')) {
                return 'status-connecting';
            }
            return 'status-disconnected';
        }
    }
};

// Make HomePage available globally
window.HomePage = HomePage;
