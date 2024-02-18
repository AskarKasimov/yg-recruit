package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Pramod-Devireddy/go-exprtk"
)

type WorkerAdding struct {
	Name string `json:"name" binding:"required"`
}

type Expression struct {
	Id           int64  `json:"id"`
	IncomingDate int64  `json:"incomingDate"`
	Vanilla      string `json:"vanilla"`
	Answer       string `json:"answer"`
	Progress     string `json:"progress"`
}

type ExpressionSolving struct {
	Id     int64  `json:"id" binding:"required"`
	Answer string `json:"answer" binding:"required"`
}

var NAME string = "33few"

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

func main() {
	ID, err := getId()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://colonel:8080/api/v1/worker/heartbeat", nil)
			if err != nil {
				log.Fatalln(err)
			}

			req.Header.Set("Authorization", strconv.FormatInt(ID, 10))

			res, err := client.Do(req)
			if err != nil {
				log.Fatalln(err)
			}
			if res.StatusCode != 200 {
				log.Fatalln("Incorrect heartbeat response")
			}

			log.Println("Successful heartbeat")

			time.Sleep(time.Minute) // Пауза на 1 минуту
		}
	}()
	for i := 0; i < 10; i++ {
		go func() {
			for {
				time.Sleep(time.Minute)
				client := &http.Client{}
				req, err := http.NewRequest("GET", "http://colonel:8080/api/v1/worker/want_to_calculate", nil)
				if err != nil {
					log.Println(err)
					continue
				}

				req.Header.Set("Authorization", strconv.FormatInt(ID, 10))

				res, err := client.Do(req)
				if err != nil {
					log.Println(err)
					continue
				}
				body, err := io.ReadAll(res.Body)
				if err != nil {
					log.Println(err)
					continue
				}
				res.Body.Close()
				var expression Expression
				err = json.Unmarshal(body, &expression)
				if err != nil {
					log.Println(err)
					continue
				}
				exprtkObj := exprtk.NewExprtk()
				exprtkObj.SetExpression(expression.Vanilla)
				err = exprtkObj.CompileExpression()
				if err != nil {
					log.Println(err)
					continue
				}
				answer := ExpressionSolving{Id: expression.Id, Answer: fmt.Sprintf("%v", exprtkObj.GetEvaluatedValue())}
				exprtkObj.Delete()
				bytesAnswer, err := json.Marshal(answer)
				if err != nil {
					log.Println(err)
					continue
				}
				solveReq, err := http.NewRequest("POST", "http://colonel:8080/api/v1/expression/solve", bytes.NewBuffer(bytesAnswer))
				if err != nil {
					log.Println(err)
					continue
				}
				solveReq.Header.Set("Authorization", strconv.FormatInt(ID, 10))

				solveResp, err := client.Do(solveReq)
				if err != nil {
					log.Println(err)
					continue
				}
				if solveResp.StatusCode != 200 {
					log.Println(err)
					continue
				}
				log.Println("Success")
			}
		}()
	}
	// Бесконечный цикл для предотвращения завершения программы
	select {}
}
