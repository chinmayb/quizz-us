package quiz

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/chinmayb/brainiac-brawl/pkg/data"
)

type Question struct {
	Quest string `json:`
}

type Answer struct {
	Ans string
}

// QuizEngine is the actual implementation of producing questions
// & validating answers
type QuizEnginer interface {

	// Retuns a channel of question to the given request
	ProduceQuestions(ctx context.Context, req any) (chan data.QuizData, chan error)

	// Validates the answer that is present in answer channel
	ValidateAnswer(ctx context.Context, ans chan data.QuizData) (<-chan bool, <-chan error)
}

type quizengine struct {
}

func NewQuizEnginer() QuizEnginer {
	return &quizengine{}
}

func (q *quizengine) ProduceQuestions(ctx context.Context, req any) (chan data.QuizData, chan error) {
	questn := make(chan data.QuizData)
	go func() {
		num := rand.Int31n(int32(len(data.QuizDataRefined)))
		quizD := data.QuizDataRefined[num]
		questn <- quizD
	}()
	return questn, nil
}

func (q *quizengine) ValidateAnswer(ctx context.Context, ans chan data.QuizData) (<-chan bool, <-chan error) {
	var (
		result chan bool
		errCh  chan error
	)
	go func() {
		for {
			select {
			case quizData, ok := <-ans:
				if !ok {
					return
				}
				if quizData.Answer == "" {
					result <- false
				}
				// TODO enhance the input to regex the answer
				// such as gavaskar can be considered as right answer
				// instead of sunil gavaskar
				if data.QuizDataRefined[quizData.Id].Answer == quizData.Answer {
					result <- true
				}
			case <-ctx.Done():
				errCh <- fmt.Errorf("context canceled")
			}
		}
	}()
	return result, errCh
}
