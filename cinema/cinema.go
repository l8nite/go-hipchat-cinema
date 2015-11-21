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
	Actor string
	Text  string
	Delay time.Duration
}

type Scene struct {
	Intro  string
	Actors map[string]string // name: color
	Lines  []Line
}

type Movie struct {
	Title  string
	Scenes []Scene
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

		if strings.EqualFold("SCENE", actor) {
			scene := Scene{Actors: make(map[string]string), Intro: fmt.Sprintf("<em>%s</em>", text)}
			movie.Scenes = append(movie.Scenes, scene)
			continue
		}

		scene := &movie.Scenes[len(movie.Scenes)-1]
		scene.Actors[actor] = randomColor() // TODO: read actors.json
		wordCount := len(strings.Split(text, " "))
		line := Line{
			Actor: actor,
			Text:  text,
			Delay: time.Duration(math.Max(float64(wordCount/4), 3)) * time.Second,
		}
		scene.Lines = append(scene.Lines, line)
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
