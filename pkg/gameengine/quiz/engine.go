package quiz

import (
	"context"
	"math/rand"
	"strconv"
	"strings"

	"github.com/chinmayb/quizz-us/pkg/data"
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
		quizD := data.QuizDataRefined[strconv.Itoa(int(num))]
		questn <- &quizD
	}()
	return questn, nil
}

func (q *quizengine) ValidateAnswer(ctx context.Context, qui *data.QuizData) bool {
	// TODO enhance the input to regex the answer
	// such as gavaskar can be considered as right answer
	// instead of sunil gavaskar
	return strings.EqualFold(strings.ToLower(data.QuizDataRefined[qui.Id].Answer), strings.ToLower(qui.Answer))
}
