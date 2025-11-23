const HostGameForm = {
    props: {
        isConnected: {
            type: Boolean,
            default: false
        },
        isHosting: {
            type: Boolean,
            default: false
        }
    },

    emits: ['start-host', 'cancel-host'],

    data() {
        return {
            playerName: '',
            gameCode: '',
            errors: {
                playerName: '',
                gameCode: ''
            }
        };
    },

    computed: {
        canHost() {
            return this.isConnected &&
                   this.playerName.trim().length >= 2 &&
                   /^[A-Z0-9]{6}$/.test(this.gameCode) &&
                   !this.isHosting;
        }
    },

    template: `
        <div class="join-game-form">
            <div class="form-overlay" @click="$emit('cancel-host')"></div>

            <div class="form-modal">
                <div class="form-header">
                    <h2 class="form-title">Host Game</h2>
                    <button class="close-btn" @click="$emit('cancel-host')">
                        <span>&times;</span>
                    </button>
                </div>

                <form @submit.prevent="handleSubmit" class="form-content">
                    <div class="form-group">
                        <label for="host-player-name" class="form-label">Your Name</label>
                        <input
                            type="text"
                            id="host-player-name"
                            v-model="playerName"
                            @input="clearError('playerName')"
                            placeholder="Enter your name"
                            class="form-input"
                            :class="{ error: errors.playerName }"
                            maxlength="20"
                            required
                        >
                        <div v-if="errors.playerName" class="error-message">
                            {{ errors.playerName }}
                        </div>
                    </div>

                    <div class="form-group">
                        <div class="form-label-row">
                            <label for="host-game-code" class="form-label">Game Code</label>
                            <button type="button" class="btn btn-tertiary" @click="generateCode" :disabled="isHosting">
                                Generate
                            </button>
                        </div>
                        <input
                            type="text"
                            id="host-game-code"
                            v-model="gameCode"
                            @input="formatGameCode"
                            placeholder="Create a game code (e.g., ABC123)"
                            class="form-input game-code-input"
                            :class="{ error: errors.gameCode }"
                            maxlength="6"
                            required
                        >
                        <div v-if="errors.gameCode" class="error-message">
                            {{ errors.gameCode }}
                        </div>
                        <div class="form-hint">
                            Share this 6-character code with players to join your game
                        </div>
                    </div>

                    <div v-if="!isConnected" class="connection-warning">
                        <span class="warning-icon">⚠️</span>
                        <span>Connecting to server...</span>
                    </div>

                    <div class="form-actions">
                        <button
                            type="button"
                            class="btn btn-secondary"
                            @click="$emit('cancel-host')"
                            :disabled="isHosting"
                        >
                            Cancel
                        </button>

                        <button
                            type="submit"
                            class="btn btn-primary"
                            :disabled="!canHost"
                        >
                            <span v-if="isHosting" class="loading-spinner"></span>
                            <span v-if="!isHosting">Start Game</span>
                            <span v-else>Starting...</span>
                        </button>
                    </div>
                </form>
            </div>
        </div>
    `,

    methods: {
        formatGameCode() {
            this.gameCode = this.gameCode.replace(/\s/g, '').toUpperCase();
            this.clearError('gameCode');
        },

        generateCode() {
            const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
            let code = '';
            for (let i = 0; i < 6; i++) {
                code += chars.charAt(Math.floor(Math.random() * chars.length));
            }
            this.gameCode = code;
            this.clearError('gameCode');
        },

        clearError(field) {
            this.errors[field] = '';
        },

        validateForm() {
            let isValid = true;
            this.errors = { playerName: '', gameCode: '' };

            if (!this.playerName.trim()) {
                this.errors.playerName = 'Player name is required';
                isValid = false;
            } else if (this.playerName.trim().length < 2) {
                this.errors.playerName = 'Player name must be at least 2 characters';
                isValid = false;
            }

            if (!this.gameCode.trim()) {
                this.errors.gameCode = 'Game code is required';
                isValid = false;
            } else if (this.gameCode.length !== 6) {
                this.errors.gameCode = 'Game code must be exactly 6 characters';
                isValid = false;
            } else if (!/^[A-Z0-9]{6}$/.test(this.gameCode)) {
                this.errors.gameCode = 'Game code must contain only letters and numbers';
                isValid = false;
            }

            if (!this.isConnected) {
                this.errors.gameCode = this.errors.gameCode || 'Connection required to host game';
                isValid = false;
            }

            return isValid;
        },

        handleSubmit() {
            if (!this.validateForm()) {
                return;
            }

            this.$emit('start-host', {
                playerName: this.playerName.trim(),
                gameCode: this.gameCode.trim()
            });
        }
    },

    mounted() {
        this.$nextTick(() => {
            const firstInput = this.$el.querySelector('#host-player-name');
            if (firstInput) {
                firstInput.focus();
            }
        });
    }
};

window.HostGameForm = HostGameForm;
