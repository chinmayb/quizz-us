const HomePage = {
    emits: ['Join-Game', 'Host-Game'],

    template: `
        <div class="home-page">
            <!-- Hero Section -->
            <div class="hero-section">
                <h1 class="main-title">QuizUS</h1>
                <p class="subtitle">
                    A Multiplayer Quiz Gaming Platform
                </p>
            </div>
            
            <!-- Action Buttons Section -->
            <div class="action-section">
                <div class="action-buttons">
                    <button 
                        class="btn btn-join"
                        @click="$emit('Join-Game')"
                    >
                        Join
                    </button>
                    
                    <button 
                        class="btn btn-host"
                        @click="$emit('Host-Game')"
                    >
                        Host
                    </button>
                </div>
            </div>
        </div>
    `
};

// Make HomePage available globally
window.HomePage = HomePage;
