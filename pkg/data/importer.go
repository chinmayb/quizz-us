package data

import (
	"io"
	"net/url"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"
)

type QuizData struct {
	Id       string
	Question string   `yaml:question`
	Answer   string   `yaml:answer`
	ImageSrc *url.URL `yaml:imageSrc`
	Tags     []string `yaml:tags`
	Hints    []string `yaml:hints`
}

// RefinedQUizdata groups by tags and serial
//
//			"1": {
//					"question":  "",
//		            "answer": "",
//						},
//	        "2": {"question": "",
//						}
//			   	}
//
// &
// {"political": [1,100,12], "sports": [2,5,100]}
var QuizDataByTag map[string][]string
var QuizDataRefined map[string]QuizData

func PopulateRefinedData(quizData []QuizData) {
	// populate by tags
	QuizDataByTag = make(map[string][]string)
	QuizDataRefined = make(map[string]QuizData)
	for _, data := range quizData {
		for _, tag := range data.Tags {
			QuizDataByTag[tag] = append(QuizDataByTag[tag], data.Id)
		}
	}

	// Convert imageSrc strings to URL
	for i, quiz := range quizData {
		if quiz.ImageSrc != nil {
			parsedURL, err := url.Parse(quiz.ImageSrc.String())
			if err != nil {
				continue
			}
			quizData[i].ImageSrc = parsedURL
		}
		quiz.Id = strconv.Itoa(i + 1)
		QuizDataRefined[quiz.Id] = quiz
	}
}

// ParseQuizData parses the YAML file into a slice of QuizData
func ParseQuizData(filename string) error {
	var quizData []QuizData

	// Read the YAML file
	f, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	// Unmarshal the YAML data
	err = yaml.Unmarshal(data, &quizData)
	if err != nil {
		return err
	}
	PopulateRefinedData(quizData)
	return nil
}
