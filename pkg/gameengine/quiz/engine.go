package quiz

import (
	"context"
	"math/rand"

	"github.com/chinmayb/brainiac-brawl/pkg/data"
)

// QuizEngine is the actual implementation of producing questions
// & validating answers
type QuizEnginer interface {

	// Retuns a channel of unique questions to the given request
	ProduceQuestions(ctx context.Context, req any) (chan *data.QuizData, chan error)

	// Validates the answer that is present in answer channel
	ValidateAnswer(ctx context.Context, ans *data.QuizData) bool
}

type quizengine struct {
}

func NewQuizEnginer() QuizEnginer {
	return &quizengine{}
}

// ProduceQuestions produce unique question
func (q *quizengine) ProduceQuestions(ctx context.Context, req any) (chan *data.QuizData, chan error) {
	questn := make(chan *data.QuizData)
	go func() {
		num := rand.Int31n(int32(len(data.QuizDataRefined)))
		quizD := data.QuizDataRefined[num]
		questn <- &quizD
	}()
	return questn, nil
}

func (q *quizengine) ValidateAnswer(ctx context.Context, ans *data.QuizData) bool {
	// TODO enhance the input to regex the answer
	// such as gavaskar can be considered as right answer
	// instead of sunil gavaskar
	return data.QuizDataRefined[ans.Id].Answer == ans.Answer
}
