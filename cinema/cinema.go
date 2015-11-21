package cinema

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

type Line struct {
	actor string
	text  string
}

type Scene struct {
	intro  string
	actors map[string]string // name: color
	lines  []Line
}

type Movie struct {
	scenes []Scene
}

func ParseMovieFile(movieName string) (*Movie, error) {
	file, err := os.Open(fmt.Sprintf("./movies/%s/script.txt", movieName))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	defer file.Close()

	movie := Movie{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tokens := strings.SplitN(scanner.Text(), ":", 2)
		actor := strings.TrimSpace(tokens[0])
		text := strings.TrimSpace(tokens[1])

		fmt.Printf("%s ---- %s", actor, text)

		if strings.EqualFold("SCENE", actor) {
			scene := Scene{actors: make(map[string]string), intro: fmt.Sprintf("<em>%s</em>", text)}
			movie.scenes = append(movie.scenes, scene)
			continue
		}

		scene := &movie.scenes[len(movie.scenes)-1]
		scene.actors[actor] = "green" // TODO: read actors.json
		scene.lines = append(scene.lines, Line{actor: actor, text: text})
	}

	/*
		for _, scene := range movie.scenes {
			for _, line := range scene.lines {
				fmt.Printf("%s: %s\n", line.actor, line.text)
			}
		}
	*/

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &movie, nil
}
