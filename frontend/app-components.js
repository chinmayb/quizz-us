const { createApp } = Vue;

createApp({
  components: {
    UserAvatars: window.UserAvatars,
    QuizQuestion: window.QuizQuestion,
    SidebarMenu: window.SidebarMenu
  },
  
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
        image: null,
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
    // Handle answer submission
    handleSubmitAnswer() {
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
    
    // Handle menu item selection
    handleSelectMenuItem(item) {
      // Remove active state from all items
      this.menuItems.forEach(menuItem => {
        menuItem.active = false;
      });
      
      // Set active state for selected item
      item.active = true;
      
      console.log('Selected menu item:', item.label);
    },
    
    // Handle question rating
    handleRateQuestion(rating) {
      this.questionRating = rating;
      console.log('Rated question:', rating, 'stars');
    },
    
    // Load a new question
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
        },
        {
          id: 4,
          title: 'What is the smallest country in the world?',
          text: 'This country is located within Rome.',
          image: null,
          correctAnswer: 'Vatican City'
        }
      ];
      
      const randomQuestion = sampleQuestions[Math.floor(Math.random() * sampleQuestions.length)];
      this.currentQuestion = randomQuestion;
      this.resetQuestion();
    }
  },
  
  mounted() {
    console.log('Quizz-Us modular component-based app initialized successfully!');
    console.log('Components loaded:', {
      UserAvatars: !!window.UserAvatars,
      QuizQuestion: !!window.QuizQuestion,
      SidebarMenu: !!window.SidebarMenu
    });
  }
}).mount('#app');
