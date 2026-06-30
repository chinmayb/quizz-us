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

  data() {
    return {
      imageFailed: false
    };
  },

  watch: {
    'question.id'() {
      // Reset image error state whenever a new question arrives
      this.imageFailed = false;
    }
  },

  template: `
    <div class="quiz-card">
      <!-- Header -->
      <div class="quiz-card__header">
        <span class="quiz-card__badge">Q{{ questionNumber }}</span>
        <span v-if="totalQuestions" class="quiz-card__progress">
          Question {{ questionNumber }} of {{ totalQuestions }}
        </span>
      </div>

      <!-- Media / Image area -->
      <div class="quiz-card__media">
        <img
          v-if="imageUrl && !imageFailed"
          :src="imageUrl"
          :alt="'Question ' + questionNumber + ' image'"
          class="quiz-card__image"
          @error="imageFailed = true"
        />
        <div v-else class="quiz-card__image-placeholder" aria-hidden="true">
          <span class="quiz-card__image-icon">🖼️</span>
          <span class="quiz-card__image-hint">{{ imageUrl ? 'Image unavailable' : 'No image for this question' }}</span>
        </div>
      </div>

      <!-- Question text -->
      <h2 class="quiz-card__question">{{ questionText }}</h2>

      <!-- Answer area -->
      <div class="quiz-card__answer">
        <label for="answer-input" class="quiz-card__label">Your answer</label>
        <div class="quiz-card__input-row">
          <input
            type="text"
            id="answer-input"
            :value="userAnswer"
            @input="$emit('update:userAnswer', $event.target.value)"
            @keyup.enter="handleSubmit"
            placeholder="Type your answer…"
            class="quiz-card__input"
            :disabled="isSubmitted"
            autocomplete="off"
          />
          <button
            @click="handleSubmit"
            class="quiz-card__submit"
            :disabled="!canSubmit"
          >
            {{ submitButtonText }}
          </button>
        </div>

        <!-- Result feedback -->
        <transition name="result-fade">
          <div v-if="showResult" :class="['quiz-card__result', resultClass]">
            <span class="quiz-card__result-icon">{{ resultIcon }}</span>
            <span class="quiz-card__result-text">{{ resultMessage }}</span>
          </div>
        </transition>

        <!-- Next question -->
        <transition name="button-fade">
          <button
            v-if="showResult"
            @click="$emit('next-question')"
            class="quiz-card__next"
          >
            Next Question →
          </button>
        </transition>
      </div>
    </div>
  `,

  computed: {
    questionText() {
      return this.question.question || this.question.text || '';
    },

    imageUrl() {
      return this.question.image || this.question.imageSrc || this.question.image_src || '';
    },

    canSubmit() {
      return this.userAnswer.trim().length > 0 && !this.isSubmitted;
    },

    submitButtonText() {
      return this.isSubmitted ? '✓ Submitted' : 'Submit';
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
      return this.isCorrect ? 'is-correct' : 'is-incorrect';
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
