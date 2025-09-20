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
            <!-- Hero Section -->
            <div class="hero-section">
                <h1 class="main-title">QuizUS</h1>
                <p class="subtitle">
                    Quiz
                </p>
            </div>
            
            <!-- Action Buttons Section -->
            <div class="action-section">
                <div class="action-buttons">
                    <button 
                        class="btn btn-join"
                        @click="$emit('join-game')"
                        :disabled="!isConnected"
                    >
                        join
                    </button>
                    
                    <button 
                        class="btn btn-host"
                        @click="$emit('host-game')"
                        :disabled="!isConnected"
                    >
                        host
                    </button>
                </div>
                
                <!-- Connection Status -->
                <div class="connection-status" :class="connectionStatusClass">
                    <div class="status-indicator"></div>
                    <span class="status-text">{{ connectionStatus }}</span>
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
