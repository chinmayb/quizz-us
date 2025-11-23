# Quizz-Us Frontend

A Vue.js frontend for the Quizz-Us application with multiple implementation approaches.

## 🚀 Quick Start

```bash
# Modular components (recommended for development)
open index-modular.html
```

## 📁 Files

| File | Description | Approach |
|------|-------------|----------|
| **`index-modular.html`** | ✅ Modular component-based app | External components |
| **`index-components.html`** | ✅ Inline component-based app | Inline components |
| **`index.html`** | Original single-file implementation | Single file |
| **`styles.css`** | Shared CSS styling | - |
| **`app.js`** | Original single-file Vue app | - |
| **`app-components.js`** | Modular component app logic | - |

## 🏗️ Component Architecture

### Modular Components (Recommended)
```
components/
├── UserAvatars.js      # User avatar display
├── QuizQuestion.js     # Question and answer handling
└── SidebarMenu.js      # Navigation and rating
```

**Benefits:**
- ✅ **Separation of Concerns** - Each component in its own file
- ✅ **Reusability** - Components can be used in other projects
- ✅ **Maintainability** - Easy to edit individual components
- ✅ **Team Development** - Multiple developers can work on different components
- ✅ **Testing** - Individual components can be tested separately

### Inline Components
```html
<script>
  const UserAvatars = { /* inline definition */ };
  const QuizQuestion = { /* inline definition */ };
  const SidebarMenu = { /* inline definition */ };
</script>
```

**Benefits:**
- ✅ **Self-contained** - Everything in one file
- ✅ **No external dependencies** - No need to load separate files
- ✅ **Simple deployment** - Just one HTML file

## 🎯 Features

- **Interactive Quiz Interface** - Answer questions with real-time feedback
- **Component Architecture** - Modular Vue.js components
- **Responsive Design** - Works on desktop and mobile
- **Star Rating System** - Rate questions for quality feedback
- **User Avatars** - Display multiple users in header
- **Sidebar Navigation** - Browse question categories

## 🎮 Usage

1. **Answer Questions**: Type your answer and click Submit
2. **Navigate**: Use sidebar menu to switch categories  
3. **Rate Questions**: Click stars to rate question quality
4. **Next Question**: Click "Next Question" for new content

## 📱 Sample Questions

- "What is the capital of France?" → Paris
- "What is 2 + 2?" → 4
- "What is the largest planet?" → Jupiter

## 🔧 Technical Details

- **Framework**: Vue.js 3 (CDN)
- **Styling**: Custom CSS with responsive design
- **Components**: Modular architecture with props/events
- **Browser Support**: Modern browsers with ES6

## 🚀 Development

### Adding Questions
```javascript
const sampleQuestions = [
  {
    id: 5,
    title: 'Your Question',
    text: 'Question description',
    correctAnswer: 'Answer'
  }
];
```

### Customizing Components
Edit individual component files in the `components/` directory:
- `UserAvatars.js` - Avatar display logic
- `QuizQuestion.js` - Question and answer handling
- `SidebarMenu.js` - Navigation and rating system

### Customizing Styles
Edit `styles.css` for colors, layout, and typography changes.

## 📋 Component Structure

```
UserAvatars    → Display user avatars in header
QuizQuestion   → Question display and answer handling  
SidebarMenu    → Navigation and rating system
```
