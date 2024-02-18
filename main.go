package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Pramod-Devireddy/go-exprtk"
	"github.com/google/uuid"
)

type WorkerAdding struct {
	Name string `json:"name"`
}

type Expression struct {
	Id           int64  `json:"id"`
	IncomingDate int64  `json:"incomingDate"`
	Vanilla      string `json:"vanilla"`
	Answer       string `json:"answer"`
	Progress     string `json:"progress"`
}

type ExpressionSolving struct {
	Id     int64  `json:"id"`
	Answer string `json:"answer"`
}

var GOROUTINES int
var NAME string = uuid.New().String()
var ID int64
var MULTIPLICATION int64
var DIVISION int64
var ADDITION int64
var SUBTRACTION int64

func init() {
	goroutines, err := strconv.Atoi(os.Getenv("GOROUTINES"))
	if err != nil {
		log.Fatalln(err)
	}
	GOROUTINES = goroutines

	multiplication, err := strconv.ParseInt(os.Getenv("MULTIPLICATION"), 10, 64)
	if err != nil {
		log.Fatalln(err)
	}
	MULTIPLICATION = multiplication

	division, err := strconv.ParseInt(os.Getenv("DIVISION"), 10, 64)
	if err != nil {
		log.Fatalln(err)
	}
	DIVISION = division

	addition, err := strconv.ParseInt(os.Getenv("ADDITION"), 10, 64)
	if err != nil {
		log.Fatalln(err)
	}
	ADDITION = addition

	subtraction, err := strconv.ParseInt(os.Getenv("SUBTRACTION"), 10, 64)
	if err != nil {
		log.Fatalln(err)
	}
	SUBTRACTION = subtraction

	log.Printf("Name (uuid): %s", NAME)
	log.Printf("Number of goroutines: %d", GOROUTINES)
	log.Printf("Time for multiplication: %d", MULTIPLICATION)
	log.Printf("Time for division: %d", DIVISION)
	log.Printf("Time for addition: %d", ADDITION)
	log.Printf("Time for subtraction: %d", SUBTRACTION)
}

func getId() (int64, error) {
	workerAdding := WorkerAdding{Name: NAME}

	client := &http.Client{}
	body, err := json.Marshal(workerAdding)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", "http://colonel:8080/api/v1/worker/register", bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	ok, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	res.Body.Close()

	id, err := strconv.ParseInt(string(ok), 10, 64)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func getExpressionToSolve(client *http.Client) (Expression, error) {
	req, err := http.NewRequest("GET", "http://colonel:8080/api/v1/worker/want_to_calculate", nil)
	if err != nil {
		return Expression{}, err
	}

	req.Header.Set("Authorization", strconv.FormatInt(ID, 10))

	res, err := client.Do(req)
	if err != nil {
		return Expression{}, err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return Expression{}, err
	}

	var expression Expression

	err = json.Unmarshal(body, &expression)
	if err != nil {
		return Expression{}, err
	}

	return expression, nil
}

func countTimeToSleep(expression string) time.Duration {
	var totalTime time.Duration = 0
	for _, char := range expression {
		if char == []rune("*")[0] {
			totalTime += time.Second * time.Duration(MULTIPLICATION)
		}
		if char == []rune("/")[0] {
			totalTime += time.Second * time.Duration(DIVISION)
		}
		if char == []rune("+")[0] {
			totalTime += time.Second * time.Duration(ADDITION)
		}
		if char == []rune("-")[0] {
			totalTime += time.Second * time.Duration(SUBTRACTION)
		}
	}
	log.Printf("Sleeping %f seconds", totalTime.Seconds())
	return totalTime
}

func solveExpression(expression Expression) (string, error) {
	exprtkObj := exprtk.NewExprtk()
	defer exprtkObj.Delete()
	exprtkObj.SetExpression(expression.Vanilla)

	err := exprtkObj.CompileExpression()
	if err != nil {
		return "", err
	}

	time.Sleep(countTimeToSleep(expression.Vanilla))

	return fmt.Sprintf("%v", exprtkObj.GetEvaluatedValue()), nil
}

func sendAnswer(client *http.Client, bytesAnswer []byte) error {
	solveReq, err := http.NewRequest("POST", "http://colonel:8080/api/v1/expression/solve", bytes.NewBuffer(bytesAnswer))
	if err != nil {
		return err
	}
	solveReq.Header.Set("Authorization", strconv.FormatInt(ID, 10))

	solveResp, err := client.Do(solveReq)
	if err != nil || solveResp.StatusCode != 200 {
		return err
	}

	return nil
}

func process() error {
	client := &http.Client{}
	expression, err := getExpressionToSolve(client)
	if err != nil {
		return err
	}

	solve, err := solveExpression(expression)
	if err != nil {
		return err
	}

	answer := ExpressionSolving{Id: expression.Id, Answer: solve}
	bytesAnswer, err := json.Marshal(answer)
	if err != nil {
		return err
	}

	err = sendAnswer(client, bytesAnswer)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	id, err := getId()
	if err != nil {
		log.Fatal(err)
	}
	ID = id

	for i := 0; i < GOROUTINES; i++ {
		go func() {
			for {
				time.Sleep(time.Minute)

				err := process()
				if err != nil {
					log.Println(err)
				} else {
					log.Println("Success")
				}
			}
		}()
	}
	// Бесконечный цикл для предотвращения завершения программы
	select {}
}
