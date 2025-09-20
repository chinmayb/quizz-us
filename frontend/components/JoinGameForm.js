const JoinGameForm = {
    props: {
        isConnected: {
            type: Boolean,
            default: false
        },
        isJoining: {
            type: Boolean,
            default: false
        }
    },
    
    emits: ['join-game', 'cancel-join'],
    
    data() {
        return {
            gameCode: '',
            playerName: '',
            errors: {
                gameCode: '',
                playerName: ''
            }
        };
    },
    
    template: `
        <div class="join-game-form">
            <div class="form-overlay" @click="$emit('cancel-join')"></div>
            
            <div class="form-modal">
                <div class="form-header">
                    <h2 class="form-title">Join Game</h2>
                    <button class="close-btn" @click="$emit('cancel-join')">
                        <span>&times;</span>
                    </button>
                </div>
                
                <form @submit.prevent="handleSubmit" class="form-content">
                    <!-- Player Name Input -->
                    <div class="form-group">
                        <label for="player-name" class="form-label">Your Name</label>
                        <input 
                            type="text" 
                            id="player-name"
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
                    
                    <!-- Game Code Input -->
                    <div class="form-group">
                        <label for="game-code" class="form-label">Game Code</label>
                        <input 
                            type="text" 
                            id="game-code"
                            v-model="gameCode"
                            @input="formatGameCode"
                            placeholder="Enter game code (e.g., ABC123)"
                            class="form-input game-code-input"
                            :class="{ error: errors.gameCode }"
                            maxlength="8"
                            required
                        >
                        <div v-if="errors.gameCode" class="error-message">
                            {{ errors.gameCode }}
                        </div>
                        <div class="form-hint">
                            Game codes are 6 characters long (letters and numbers)
                        </div>
                    </div>
                    
                    <!-- Connection Status -->
                    <div v-if="!isConnected" class="connection-warning">
                        <span class="warning-icon">⚠️</span>
                        <span>Connecting to server...</span>
                    </div>
                    
                    <!-- Form Actions -->
                    <div class="form-actions">
                        <button 
                            type="button" 
                            class="btn btn-secondary"
                            @click="$emit('cancel-join')"
                            :disabled="isJoining"
                        >
                            Cancel
                        </button>
                        
                        <button 
                            type="submit" 
                            class="btn btn-primary"
                            :disabled="!canJoin"
                        >
                            <span v-if="isJoining" class="loading-spinner"></span>
                            <span v-if="!isJoining">Join Game</span>
                            <span v-else>Joining...</span>
                        </button>
                    </div>
                </form>
            </div>
        </div>
    `,
    
    computed: {
        canJoin() {
            return this.isConnected && 
                   this.playerName.trim().length > 0 && 
                   this.gameCode.trim().length >= 6 && 
                   !this.isJoining;
        }
    },
    
    methods: {
        formatGameCode() {
            // Remove spaces and convert to uppercase
            this.gameCode = this.gameCode.replace(/\s/g, '').toUpperCase();
            this.clearError('gameCode');
        },
        
        clearError(field) {
            this.errors[field] = '';
        },
        
        validateForm() {
            let isValid = true;
            
            // Reset errors
            this.errors = {
                gameCode: '',
                playerName: ''
            };
            
            // Validate player name
            if (!this.playerName.trim()) {
                this.errors.playerName = 'Player name is required';
                isValid = false;
            } else if (this.playerName.trim().length < 2) {
                this.errors.playerName = 'Player name must be at least 2 characters';
                isValid = false;
            }
            
            // Validate game code
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
            
            return isValid;
        },
        
        handleSubmit() {
            if (!this.validateForm()) {
                return;
            }
            
            this.$emit('join-game', {
                playerName: this.playerName.trim(),
                gameCode: this.gameCode.trim()
            });
        },
        
        reset() {
            this.gameCode = '';
            this.playerName = '';
            this.errors = {
                gameCode: '',
                playerName: ''
            };
        }
    },
    
    mounted() {
        // Focus on the first input when the form is shown
        this.$nextTick(() => {
            const firstInput = this.$el.querySelector('#player-name');
            if (firstInput) {
                firstInput.focus();
            }
        });
    }
};

// Make JoinGameForm available globally
window.JoinGameForm = JoinGameForm;
