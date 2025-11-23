// QuizQuestion Component
window.QuizQuestion = {
  props: {
    question: {
      type: Object,
      required: true
    },
    questionNumber: {
      type: Number,
      default: 1
    },
    totalQuestions: {
      type: Number,
      default: null
    },
    userAnswer: {
      type: String,
      default: ''
    },
    isSubmitted: {
      type: Boolean,
      default: false
    },
    showResult: {
      type: Boolean,
      default: false
    },
    isCorrect: {
      type: Boolean,
      default: false
    }
  },
  
  emits: ['update:userAnswer', 'submit-answer', 'next-question'],
  
  template: `
    <div class="quiz-question-card">
      <!-- Question Header -->
      <div class="question-header">
        <div class="question-number-section">
          <h2 class="question-number">Question #{{ questionNumber }}</h2>
          <p v-if="totalQuestions" class="question-progress">of {{ totalQuestions }} questions</p>
        </div>
      </div>
      
      <div class="question-divider"></div>
      
      <!-- Question Content -->
      <div class="question-body">
        <div class="question-text-container">
          <p class="question-text">{{ question.question || question.text }}</p>
        </div>
        
        <!-- Optional Question Image -->
        <div v-if="question.image" class="question-image-container">
          <img :src="question.image" :alt="'Question ' + questionNumber" class="question-image" />
        </div>
      </div>
      
      <!-- Answer Section -->
      <div class="answer-section">
        <label for="answer-input" class="answer-label">Your Answer</label>
        <div class="answer-input-group">
          <input 
            type="text" 
            id="answer-input"
            :value="userAnswer"
            @input="$emit('update:userAnswer', $event.target.value)"
            @keyup.enter="handleSubmit"
            placeholder="Type your answer here..."
            class="answer-input-field"
            :disabled="isSubmitted"
            :class="{ 'disabled': isSubmitted }"
          >
          <button 
            @click="handleSubmit" 
            class="submit-btn" 
            :disabled="!canSubmit"
            :class="{ 'disabled': !canSubmit }"
          >
            {{ submitButtonText }}
          </button>
        </div>
        
        <!-- Result Display -->
        <transition name="result-fade">
          <div v-if="showResult" :class="['result-feedback', resultClass]">
            <span class="result-icon">{{ resultIcon }}</span>
            <span class="result-text">{{ resultMessage }}</span>
          </div>
        </transition>
        
        <!-- Next Question Button -->
        <transition name="button-fade">
          <div v-if="showResult" class="next-question-container">
            <button @click="$emit('next-question')" class="next-question-btn">
              Next Question →
            </button>
          </div>
        </transition>
      </div>
    </div>
  `,
  
  computed: {
    canSubmit() {
      return this.userAnswer.trim().length > 0 && !this.isSubmitted;
    },
    
    submitButtonText() {
      if (this.isSubmitted) return '✓ Submitted';
      return 'Submit';
    },
    
    resultMessage() {
      if (!this.showResult) return '';
      if (this.isCorrect) return 'Correct! Well done!';
      const correctAnswer = this.question.correctAnswer || this.question.answer;
      return correctAnswer ? `The correct answer is: ${correctAnswer}` : 'Incorrect';
    },
    
    resultIcon() {
      return this.isCorrect ? '🎉' : '❌';
    },
    
    resultClass() {
      if (!this.showResult) return '';
      return this.isCorrect ? 'correct' : 'incorrect';
    }
  },
  
  methods: {
    handleSubmit() {
      if (this.canSubmit) {
        this.$emit('submit-answer');
      }
    }
  }
};
