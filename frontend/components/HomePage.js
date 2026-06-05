const HomePage = {
    emits: ['join-game', 'host-game'],

    template: `
        <div class="home-page">
            <div class="hero-section">
                <h1 class="main-title">QUIZZ<span class="accent-dot">·</span>US</h1>
                <p class="subtitle">multiplayer trivia &mdash; play with friends</p>
            </div>

            <div class="action-section">
                <div class="action-buttons">
                    <button class="btn btn-join" @click="$emit('join-game')">Join</button>
                    <button class="btn btn-host" @click="$emit('host-game')">Host</button>
                </div>
            </div>
        </div>
    `
};

window.HomePage = HomePage;
