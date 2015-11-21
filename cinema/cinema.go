package cinema

import (
	"bufio"
	"fmt"
	"github.com/l8nite/hipchat-cinema/util"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"
)

var colors = []string{"gray", "green", "purple", "red", "yellow"}

type Line struct {
	actor string
	text  string
	delay time.Duration
}

type Scene struct {
	intro  string
	actors map[string]string // name: color
	lines  []Line
}

type Movie struct {
	Title  string
	scenes []Scene
}

func ParseMovieFile(movieName string) (*Movie, error) {
	file, err := os.Open(fmt.Sprintf("./movies/%s/script.txt", movieName))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	defer file.Close()

	movie := Movie{
		Title: util.MovieTitle(movieName),
	}

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
		scene.actors[actor] = randomColor() // TODO: read actors.json
		wordCount := len(strings.Split(text, " "))
		line := Line{
			actor: actor,
			text:  text,
			delay: time.Duration(math.Max(float64(wordCount/4), 3)) * time.Second,
		}
		scene.lines = append(scene.lines, line)
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

func randomColor() string {
	return colors[rand.Intn(len(colors))]
}
