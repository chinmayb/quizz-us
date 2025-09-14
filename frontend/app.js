
const { createApp } = Vue;

createApp({
    data() {
        return {
            // Sample users for avatar display
            users: [
                { id: 1, name: 'John Doe', initials: 'JD', avatar: null },
                { id: 2, name: 'Jane Smith', initials: 'JS', avatar: null },
                { id: 3, name: 'Bob Wilson', initials: 'BW', avatar: null },
                { id: 4, name: 'Alice Brown', initials: 'AB', avatar: null }
            ],
            
            // Current question data
            currentQuestion: {
                id: 1,
                title: 'What is the capital of France?',
                text: 'This is a sample quiz question. Can you identify the correct answer?',
                image: null, // You can add an image URL here
                correctAnswer: 'Paris'
            },
            
            // User's current answer
            userAnswer: '',
            
            // Menu items for sidebar
            menuItems: [
                { 
                    id: 1, 
                    label: 'Favorites', 
                    description: 'Your favorite questions',
                    icon: '★',
                    shortcut: null,
                    active: false
                },
                { 
                    id: 2, 
                    label: 'All Questions', 
                    description: 'Browse all available questions',
                    icon: '📚',
                    shortcut: '⇧A',
                    active: true
                },
                { 
                    id: 3, 
                    label: 'Recent', 
                    description: 'Recently viewed questions',
                    icon: '🕒',
                    shortcut: null,
                    active: false
                }
            ],
            
            // Question rating
            questionRating: 0,
            
            // Quiz state
            isSubmitted: false,
            showResult: false,
            isCorrect: false
        }
    },
    
    methods: {
        // Submit answer
        submitAnswer() {
            if (!this.userAnswer.trim()) return;
            
            this.isSubmitted = true;
            this.isCorrect = this.userAnswer.toLowerCase().trim() === this.currentQuestion.correctAnswer.toLowerCase();
            this.showResult = true;
            
            // Show result for 3 seconds, then reset
            setTimeout(() => {
                this.resetQuestion();
            }, 3000);
        },
        
        // Reset question for next attempt
        resetQuestion() {
            this.userAnswer = '';
            this.isSubmitted = false;
            this.showResult = false;
            this.isCorrect = false;
            this.questionRating = 0;
        },
        
        // Select menu item
        selectMenuItem(item) {
            // Remove active state from all items
            this.menuItems.forEach(menuItem => {
                menuItem.active = false;
            });
            
            // Set active state for selected item
            item.active = true;
            
            // Here you would typically load different questions based on selection
            console.log('Selected menu item:', item.label);
        },
        
        // Rate the current question
        rateQuestion(rating) {
            this.questionRating = rating;
            console.log('Rated question:', rating, 'stars');
        },
        
        // Load a new question (placeholder for future functionality)
        loadNewQuestion() {
            const sampleQuestions = [
                {
                    id: 2,
                    title: 'What is 2 + 2?',
                    text: 'A simple math question to test your knowledge.',
                    image: null,
                    correctAnswer: '4'
                },
                {
                    id: 3,
                    title: 'What is the largest planet in our solar system?',
                    text: 'Think about the planets and their relative sizes.',
                    image: null,
                    correctAnswer: 'Jupiter'
                }
            ];
            
            const randomQuestion = sampleQuestions[Math.floor(Math.random() * sampleQuestions.length)];
            this.currentQuestion = randomQuestion;
            this.resetQuestion();
        }
    },
    
    computed: {
        // Computed property for submit button state
        canSubmit() {
            return this.userAnswer.trim().length > 0 && !this.isSubmitted;
        },
        
        // Computed property for result message
        resultMessage() {
            if (!this.showResult) return '';
            return this.isCorrect ? 'Correct! 🎉' : `Incorrect. The answer is: ${this.currentQuestion.correctAnswer}`;
        },
        
        // Computed property for result class
        resultClass() {
            if (!this.showResult) return '';
            return this.isCorrect ? 'result-correct' : 'result-incorrect';
        }
    },
    
    mounted() {
        // Initialize the app
        console.log('Quizz-Us app initialized');
        
        // You can add any initialization logic here
        // For example, loading questions from an API
    }
}).mount('#app');
