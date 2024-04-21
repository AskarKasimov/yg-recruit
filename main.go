package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Pramod-Devireddy/go-exprtk"
	"github.com/google/uuid"
)

type WorkerAdding struct {
	Name               string `json:"name"`
	NumberOfGoroutines int    `json:"number_of_goroutines"`
}

type Expression struct {
	Id           uuid.UUID `json:"id"`
	IncomingDate int64     `json:"incomingDate"`
	Vanilla      string    `json:"vanilla"`
	Answer       string    `json:"answer"`
	Progress     string    `json:"progress"`
}

type ExpressionSolving struct {
	Id     uuid.UUID `json:"id"`
	Answer string    `json:"answer"`
}

type Configuration struct {
	Name string
}

var GOROUTINES int
var OWN_NAME string
var ID_FROM_SERVER string
var MULTIPLICATION int64
var DIVISION int64
var ADDITION int64
var SUBTRACTION int64

var mutex sync.Mutex
var serverBroken bool = false
var channel = make(chan int)

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

	confFile, _ := os.Open("conf/conf.json")
	defer confFile.Close()
	decoder := json.NewDecoder(confFile)
	configuration := Configuration{}
	err = decoder.Decode(&configuration)
	if err != nil {
		log.Printf("%s: %s\n%s", "Error parsing ./conf/conf.json", err, "So creating the new one...")

		newConfFile, _ := os.Create(`conf/conf.json`)
		defer newConfFile.Close()

		OWN_NAME = uuid.New().String()

		newConf := Configuration{Name: OWN_NAME}
		jsonnedBytes, _ := json.Marshal(newConf)
		newConfFile.Write(jsonnedBytes)
	} else {
		OWN_NAME = configuration.Name
	}

	log.Printf("Name (uuid): %s", OWN_NAME)
	log.Printf("Number of goroutines: %d", GOROUTINES)
	log.Printf("Time (seconds) for multiplication: %d", MULTIPLICATION)
	log.Printf("Time (seconds) for division: %d", DIVISION)
	log.Printf("Time (seconds) for addition: %d", ADDITION)
	log.Printf("Time (seconds) for subtraction: %d", SUBTRACTION)
}

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func getId() (string, error) {
	workerAdding := WorkerAdding{Name: OWN_NAME, NumberOfGoroutines: GOROUTINES}

	client := &http.Client{}
	body, err := json.Marshal(workerAdding)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "http://colonel:8080/api/v1/worker/register", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	ok, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	res.Body.Close()

	if IsValidUUID(string(ok)) {
		return string(ok), nil
	}

	return "", errors.New("Not valid id got from colonel")
}

func getExpressionToSolve(client *http.Client) (Expression, error) {
	req, err := http.NewRequest("GET", "http://colonel:8080/api/v1/worker/want_to_calculate", nil)
	if err != nil {
		return Expression{}, err
	}

	req.Header.Set("Authorization", ID_FROM_SERVER)

	res, err := client.Do(req)
	if err != nil {
		mutex.Lock()
		serverBroken = true
		mutex.Unlock()
		return Expression{}, nil
	} else {
		mutex.Lock()
		serverBroken = false
		mutex.Unlock()
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
	solveReq.Header.Set("Authorization", ID_FROM_SERVER)

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
	ID_FROM_SERVER = id

	for i := 0; i < GOROUTINES; i++ {
		go func() {
			for {
				err := process()
				if err != nil {
					log.Println(err)
				} else {
					log.Println("Success")
				}
				mutex.Lock()
				if serverBroken {
					mutex.Unlock()
					time.Sleep(time.Second * 2)
				} else {
					mutex.Unlock()
					time.Sleep(time.Minute / 2)
				}
			}
		}()
	}
	// Бесконечный цикл для предотвращения завершения программы
	select {}
}
