const ScoreDisplay = {
    props: {
        stats: {
            type: Object,
            default: () => ({
                sampleQuestions: 1250,
                manualQuizzes: 320,
                totalScores: 2570,
                playersOnline: 45,
                gamesActive: 12
            })
        },
        animated: {
            type: Boolean,
            default: true
        }
    },
    
    data() {
        return {
            displayStats: {
                sampleQuestions: 0,
                manualQuizzes: 0,
                totalScores: 0,
                playersOnline: 0,
                gamesActive: 0
            },
            isVisible: false
        };
    },
    
    template: `
        <div class="score-display" :class="{ 'animate-in': isVisible }">
            <div class="stats-grid">
                <div class="stat-item sample-questions">
                    <div class="stat-icon">📚</div>
                    <div class="stat-content">
                        <div class="stat-number">{{ formatNumber(displayStats.sampleQuestions) }}</div>
                        <div class="stat-label">Sample Questions</div>
                        <div class="stat-description">Ready-to-use quiz questions</div>
                    </div>
                </div>
                
                <div class="stat-item manual-quizzes">
                    <div class="stat-icon">✏️</div>
                    <div class="stat-content">
                        <div class="stat-number">{{ formatNumber(displayStats.manualQuizzes) }}</div>
                        <div class="stat-label">Manual Quizzes</div>
                        <div class="stat-description">Custom quizzes created by users</div>
                    </div>
                </div>
                
                <div class="stat-item total-scores">
                    <div class="stat-icon">🏆</div>
                    <div class="stat-content">
                        <div class="stat-number">{{ formatNumber(displayStats.totalScores) }}</div>
                        <div class="stat-label">Total Scores</div>
                        <div class="stat-description">Games completed by players</div>
                    </div>
                </div>
                
                <div class="stat-item players-online">
                    <div class="stat-icon">👥</div>
                    <div class="stat-content">
                        <div class="stat-number">{{ formatNumber(displayStats.playersOnline) }}</div>
                        <div class="stat-label">Players Online</div>
                        <div class="stat-description">Currently active players</div>
                    </div>
                </div>
                
                <div class="stat-item games-active">
                    <div class="stat-icon">🎮</div>
                    <div class="stat-content">
                        <div class="stat-number">{{ formatNumber(displayStats.gamesActive) }}</div>
                        <div class="stat-label">Active Games</div>
                        <div class="stat-description">Games in progress right now</div>
                    </div>
                </div>
            </div>
            
            <!-- Performance Comparison -->
            <div class="performance-section">
                <h3 class="performance-title">Quiz Performance</h3>
                <div class="performance-bars">
                    <div class="performance-item">
                        <div class="performance-label">
                            <span>Sample Questions</span>
                            <span class="performance-value">{{ calculatePercentage('sample') }}%</span>
                        </div>
                        <div class="progress-bar">
                            <div 
                                class="progress-fill sample" 
                                :style="{ width: calculatePercentage('sample') + '%' }"
                            ></div>
                        </div>
                    </div>
                    
                    <div class="performance-item">
                        <div class="performance-label">
                            <span>Manual Quizzes</span>
                            <span class="performance-value">{{ calculatePercentage('manual') }}%</span>
                        </div>
                        <div class="progress-bar">
                            <div 
                                class="progress-fill manual" 
                                :style="{ width: calculatePercentage('manual') + '%' }"
                            ></div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `,
    
    methods: {
        formatNumber(num) {
            if (num >= 1000000) {
                return (num / 1000000).toFixed(1) + 'M';
            } else if (num >= 1000) {
                return (num / 1000).toFixed(1) + 'K';
            }
            return num.toString();
        },
        
        calculatePercentage(type) {
            const total = this.stats.sampleQuestions + this.stats.manualQuizzes;
            if (total === 0) return 0;
            
            if (type === 'sample') {
                return Math.round((this.stats.sampleQuestions / total) * 100);
            } else if (type === 'manual') {
                return Math.round((this.stats.manualQuizzes / total) * 100);
            }
            return 0;
        },
        
        animateNumbers() {
            if (!this.animated) {
                this.displayStats = { ...this.stats };
                return;
            }
            
            const duration = 2000; // 2 seconds
            const steps = 60; // 60 fps
            const stepDuration = duration / steps;
            
            const startValues = { ...this.displayStats };
            const targetValues = { ...this.stats };
            
            let currentStep = 0;
            
            const animate = () => {
                currentStep++;
                const progress = currentStep / steps;
                const easeProgress = this.easeOutQuart(progress);
                
                Object.keys(targetValues).forEach(key => {
                    const startValue = startValues[key];
                    const targetValue = targetValues[key];
                    const currentValue = startValue + (targetValue - startValue) * easeProgress;
                    this.displayStats[key] = Math.round(currentValue);
                });
                
                if (currentStep < steps) {
                    setTimeout(animate, stepDuration);
                }
            };
            
            animate();
        },
        
        easeOutQuart(t) {
            return 1 - Math.pow(1 - t, 4);
        },
        
        updateStats(newStats) {
            if (this.animated) {
                this.animateNumbers();
            } else {
                this.displayStats = { ...newStats };
            }
        }
    },
    
    watch: {
        stats: {
            handler() {
                this.animateNumbers();
            },
            deep: true
        }
    },
    
    mounted() {
        // Trigger entrance animation
        this.$nextTick(() => {
            this.isVisible = true;
            this.animateNumbers();
        });
    }
};

// Make ScoreDisplay available globally
window.ScoreDisplay = ScoreDisplay;
