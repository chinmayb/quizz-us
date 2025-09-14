// QuizQuestion Component
window.QuizQuestion = {
  props: {
    question: {
      type: Object,
      required: true
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
    <div class="question-container">
      <h2 class="question-title">{{ question.title }}</h2>
      <div class="question-content">
        <div class="question-image">
          <img :src="question.image" :alt="question.title" v-if="question.image">
          <div class="image-placeholder" v-else>
            <span>📷 Question Image</span>
          </div>
        </div>
        <div class="question-text">
          <p>{{ question.text }}</p>
        </div>
      </div>
      
      <!-- Answer Input -->
      <div class="answer-section">
        <label for="answer-input" class="input-label">Your Answer</label>
        <div class="input-container">
          <input 
            type="text" 
            id="answer-input"
            :value="userAnswer"
            @input="$emit('update:userAnswer', $event.target.value)"
            @keyup.enter="$emit('submit-answer')"
            placeholder="Type your answer here..."
            class="answer-input"
            :disabled="isSubmitted"
          >
          <button 
            @click="$emit('submit-answer')" 
            class="submit-button" 
            :disabled="!canSubmit"
          >
            {{ isSubmitted ? 'Submitted' : 'Submit' }}
          </button>
        </div>
        
        <!-- Result Display -->
        <div v-if="showResult" :class="['result-message', resultClass]">
          {{ resultMessage }}
        </div>
        
        <!-- Next Question Button -->
        <div v-if="showResult" class="next-question-section">
          <button @click="$emit('next-question')" class="next-button">
            Next Question
          </button>
        </div>
      </div>
    </div>
  `,
  
  computed: {
    canSubmit() {
      return this.userAnswer.trim().length > 0 && !this.isSubmitted;
    },
    
    resultMessage() {
      if (!this.showResult) return '';
      return this.isCorrect ? 'Correct! 🎉' : `Incorrect. The answer is: ${this.question.correctAnswer}`;
    },
    
    resultClass() {
      if (!this.showResult) return '';
      return this.isCorrect ? 'result-correct' : 'result-incorrect';
    }
  }
};
