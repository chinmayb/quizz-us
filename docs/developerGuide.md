### Frontend Structure:

```
docs/                       # all the needed docs (hopefully)
frontend/
├── index-modular.html      # 🚀 MAIN APP (Component-based)
├── home-styles.css         # Modern styling for home page
├── styles.css              # Global styles for quiz interface
├── components/             # 📦 Modular Vue Components
│   ├── HomePage.js         # Landing page with join/host buttons
│   ├── JoinGameForm.js     # Modal for joining games
│   ├── QuizQuestion.js     # Question display & answering
│   ├── ScoreDisplay.js     # Score tracking component
│   ├── SidebarMenu.js      # Navigation & rating system
│   └── UserAvatars.js      # User avatar display
└── README.md               # 📚 Comprehensive documentation
```

### Frontend development

To start the frontend development server,  run:

```
make run-frontend
```
This will launch the development server, and you can access the application in your web browser at `http://localhost:3001`
