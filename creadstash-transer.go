package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type CommandArgs struct {
	to   string
	from string
}

type CredstashDatam struct {
	key   *[]byte
	value *[]byte
}

func (cd *CredstashDatam) keyValue() string {
	return cd.keyString() + ": " + cd.valueString()
}

func (cd *CredstashDatam) keyString() string {
	if cd.key == nil {
		return ""
	}

	return string(*(cd.key))
}

func (cd *CredstashDatam) valueString() string {
	if cd.value == nil {
		return ""
	}

	return string(*(cd.value))
}

func getArguments(args *[]string) (*CommandArgs, error) {
	to := 0
	from := 0

	for i, arg := range *args {
		if arg == "--to" {
			to = i + 1
		} else if arg == "--from" {
			from = i + 1
		}
	}

	if to == 0 || from == 0 || to == from-1 || from >= len(*args) || from == to-1 || to >= len(*args) {
		return nil, errors.New("invalid command line arguments")
	}

	cArgs := CommandArgs{(*args)[to], (*args)[from]}
	return &cArgs, nil
}

func getLines(data *[]byte) *[][]byte {
	lines := make([][]byte, 0)
	var currentLine *[]byte
	for _, b := range *data {
		if currentLine == nil {
			l := make([]byte, 0)
			currentLine = &l
		}

		if b == '\n' {
			lines = append(lines, *currentLine)
			currentLine = nil
		} else {
			*currentLine = append(*currentLine, b)
		}
	}

	return &lines
}

func handleLine(data *[]byte) (*CredstashDatam, error) {
	target := -1
	qc := 0
	for i, d := range *data {
		if d == '"' {
			qc += 1
		}

		if qc < 2 {
			continue
		}

		if d == ':' {
			target = i
			break
		}
	}

	if target == -1 {
		return nil, errors.New("cannot process line")
	}

	d1 := (*data)[:target]
	d2 := (*data)[target:]

	k, err1 := removeQuotes(&d1)
	v, err2 := removeQuotes(&d2)
	if err1 != nil || err2 != nil {
		return nil, errors.New("cannot process line")
	}

	return &CredstashDatam{k, v}, nil
}

func removeQuotes(data *[]byte) (*[]byte, error) {
	lt := -1
	rt := -1
	ri := len(*data) - 1
	for i, d := range *data {
		if d == '"' && lt == -1 {
			lt = i
		}

		if (*data)[ri] == '"' && rt == -1 {
			rt = ri
		}

		ri -= 1
	}

	if lt == -1 || rt == -1 || lt == rt {
		fmt.Println("Failed to remove quotes from:", string(*data))
		return nil, errors.New("no quotes")
	}

	d := (*data)[lt+1 : rt]
	return &d, nil
}

func fillCredstashData(data *[]byte) *[]*CredstashDatam {
	lines := getLines(data)
	cdata := make([]*CredstashDatam, 0)
	for _, line := range *lines {
		cd, err := handleLine(&line)
		if err != nil {
			fmt.Println("failed to process line:", string(line))
			continue
		}

		cdata = append(cdata, cd)
	}

	return &cdata
}

func getData(cArgs *CommandArgs) (*[]byte, error) {
	cmd := exec.Command("credstash", "-t", cArgs.from, "getall")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Failed to get credstash data from", cArgs.from, err)
		return nil, errors.New("failed to get credstash data")
	}

	return &output, nil
}

func setDatum(cArgs *CommandArgs, cd *CredstashDatam) (*[]byte, error) {
	cmd := exec.Command("credstash", "-t", cArgs.to, "put", cd.keyString(), cd.valueString())
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Failed to set:", cd.keyString(), "in:", cArgs.to, err, "is it already set?")
		return &output, errors.New("failed to set credstash data")
	}

	return &output, nil
}

func deleteDatum(env string, cd *CredstashDatam) (*[]byte, error) {
	cmd := exec.Command("credstash", "-t", env, "delete", cd.keyString())
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Failed to delete:", cd.keyString(), "in:", env, err)
		return &output, errors.New("failed to delete credstash data")
	}

	return &output, nil
}

func main() {
	commandArgs := os.Args
	cArgs, err := getArguments(&commandArgs)
	if err != nil {
		os.Exit(2)
	}

	fData, err := getData(cArgs)
	if err != nil {
		os.Exit(2)
	}

	cd := fillCredstashData(fData)
	for _, d := range *cd {
		deleteDatum(cArgs.to, d)
		message, err := setDatum(cArgs, d)
		if err != nil {
			continue
		}

		fmt.Println(d.keyString(), "stored in:", cArgs.to, "message:", string(*(message)))
	}
}
