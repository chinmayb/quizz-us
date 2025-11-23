# Quizz-Us

A multiplayer quiz game platform.

## What is it?

Quizz-Us lets you host and play quiz games with multiple players in real-time. Create your own questions or use the provided quiz data.

## Quick Start

**Prerequisites:**
- Go 1.20+
- Node.js

**Run the game:**

1. Start the backend:
   ```bash
   make run-backend
   ```

2. Start the frontend (new terminal):
   ```bash
   make run-frontend
   ```

3. Open your browser and navigate to the URL shown.

## How to Play

**Players:** Join using a game code, answer questions, and see your score in real-time.

**Hosts:** Start a server, share the game code, and manage the quiz session.

## Custom Questions

Edit `quiz-data.yaml` to add your own questions:

```yaml
- question: "Your question here?"
  answer: "The answer"
  tags: ["category"]
```

## Documentation

**For Developers:**
- [Developer Guide](./docs/developerGuide.md)
- [Developer README](./README_DEV.md)
- [Admin Guide](./docs/admin.md)

**Technical Design:**
- [Game Engine](./docs/engine.md)
- [Architecture Diagram](./docs/multiplayerquizz-Page-1.drawio.svg)
- [Game Handler Flow](./docs/multiplayerquizz-Page-3-game-handler.drawio.svg)
- [Requirements](./docs/requirements.md)
- [Frontend](./frontend/README.md)
