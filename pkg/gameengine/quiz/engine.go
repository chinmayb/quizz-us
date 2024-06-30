package gameengine

type Question struct {
	Quest string
}

type Answer struct {
	Ans string
}

// QuizEngine is the actual implementation of producing questions
// & validating answers
type QuizEngine interface {

	// Retuns a channel of question to the given request
	ProduceQuestions(req any) (chan Question, error)

	// Validates the answer that is present in answer channel
	ValidateAnswer(ans chan Answer) (bool, error)
}
