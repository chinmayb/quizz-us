const HostGameForm = {
    props: {
        isHosting: {
            type: Boolean,
            default: false
        }
    },

    emits: ['start-host', 'cancel-host'],

    data() {
        return {
            playerName: '',
            gameKindId: 'quiz',
            categoriesInput: '',
            questionDurationSeconds: 30,
            targetScore: 10,
            errors: {
                playerName: '',
                questionDurationSeconds: '',
                targetScore: ''
            }
        };
    },

    computed: {
        canHost() {
            return this.playerName.trim().length >= 2 && !this.isHosting;
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
                        <label for="host-game-kind" class="form-label">Game Type</label>
                        <select
                            id="host-game-kind"
                            v-model="gameKindId"
                            class="form-input"
                            :disabled="isHosting"
                        >
                            <option value="quiz">Quiz</option>
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="host-categories" class="form-label">Categories (optional)</label>
                        <input
                            type="text"
                            id="host-categories"
                            v-model="categoriesInput"
                            placeholder="history, science, sports"
                            class="form-input"
                            :disabled="isHosting"
                        >
                    </div>

                    <div class="form-group">
                        <label for="host-duration" class="form-label">Question Duration (seconds)</label>
                        <input
                            type="number"
                            id="host-duration"
                            v-model.number="questionDurationSeconds"
                            min="5"
                            max="120"
                            class="form-input"
                            :class="{ error: errors.questionDurationSeconds }"
                            :disabled="isHosting"
                            required
                        >
                        <div v-if="errors.questionDurationSeconds" class="error-message">
                            {{ errors.questionDurationSeconds }}
                        </div>
                    </div>

                    <div class="form-group">
                        <label for="host-target-score" class="form-label">Target Score</label>
                        <input
                            type="number"
                            id="host-target-score"
                            v-model.number="targetScore"
                            min="1"
                            max="100"
                            class="form-input"
                            :class="{ error: errors.targetScore }"
                            :disabled="isHosting"
                            required
                        >
                        <div v-if="errors.targetScore" class="error-message">
                            {{ errors.targetScore }}
                        </div>
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
        clearError(field) {
            this.errors[field] = '';
        },

        validateForm() {
            let isValid = true;
            this.errors = {
                playerName: '',
                questionDurationSeconds: '',
                targetScore: ''
            };

            if (!this.playerName.trim()) {
                this.errors.playerName = 'Player name is required';
                isValid = false;
            } else if (this.playerName.trim().length < 2) {
                this.errors.playerName = 'Player name must be at least 2 characters';
                isValid = false;
            }

            if (!Number.isFinite(this.questionDurationSeconds) || this.questionDurationSeconds < 5 || this.questionDurationSeconds > 120) {
                this.errors.questionDurationSeconds = 'Question duration must be between 5 and 120 seconds';
                isValid = false;
            }

            if (!Number.isFinite(this.targetScore) || this.targetScore < 1 || this.targetScore > 100) {
                this.errors.targetScore = 'Target score must be between 1 and 100';
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
                gameKindId: this.gameKindId,
                categories: this.categoriesInput
                    .split(',')
                    .map((value) => value.trim())
                    .filter((value) => value.length > 0),
                questionDurationSeconds: this.questionDurationSeconds,
                targetScore: this.targetScore
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
