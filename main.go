package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gonum/matrix/mat64"
)

var graph = flag.Bool("graph", false, "print the graphviz dot script of match relationships")
var demo = flag.Bool("demo", false, "just use demo tournament matches")

func main() {
	flag.Parse()
	log.SetFlags(0)

	var tourn Tournament

	if *demo {
		tourn = Tournament{
			{"bob-r", "joe-c"},
			{"bob-r", "tim-c"},
			{"bob-r", "tim-c"},
			{"bob-r", "tim-c"},
			{"joe-r", "tim-c"},
			{"joe-r", "bob-c"},
			{"tim-r", "joe-c"},
			{"tim-r", "bob-c"},
			{"bob-c", "joe-r"},
			{"bob-c", "tim-r"},
			{"joe-c", "tim-r"},
			{"joe-c", "bob-r"},
			{"tim-c", "joe-r"},
			{"tim-c", "bob-r"},
		}
	} else {
		matches, err := ParseMatches(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		tourn = Tournament(matches)
	}

	m := tourn.Matrix()

	if *graph {
		Graph(os.Stdout, m, tourn.Players())
		return
	}

	prefix := "tournmat = "
	fmt.Printf("%v%v\n", prefix, mat64.Formatted(m, mat64.Prefix(strings.Repeat(" ", len(prefix)))))

	fmt.Println()
	eig := mat64.Eigen(m, 1e-10)
	prefix = "eigvects = "
	fmt.Printf("%v%.4v\n", prefix, mat64.Formatted(eig.V, mat64.Prefix(strings.Repeat(" ", len(prefix)))))

	fmt.Println()
	prefix = "eigvals = "
	fmt.Printf("%v%.4v\n", prefix, mat64.Formatted(eig.D(), mat64.Prefix(strings.Repeat(" ", len(prefix)))))

}

func ParseMatches(r io.Reader) ([]Match, error) {
	s := bufio.NewScanner(os.Stdin)
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		i := bytes.IndexAny(data, "\n \t\r")
		if len(data) == 0 {
			return 0, nil, nil
		} else if i == -1 {
			return len(data), data, nil
		}

		token = bytes.TrimSpace(data[:i])
		if len(token) == 0 {
			token = nil
		}
		return i + 1, token, nil
	})

	matches := []Match{}
	for {
		if !s.Scan() {
			break
		}
		winner := s.Text()
		if !s.Scan() {
			return nil, errors.New("odd number of players causes opponentless match")
		}
		loser := s.Text()

		matches = append(matches, Match{winner, loser})
	}
	return matches, s.Err()
}

func Graph(w io.Writer, m *mat64.Dense, players map[int]string) {
	fmt.Fprintf(w, "digraph matches {\n")
	r, c := m.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			if v := m.At(i, j); v > 0 {
				winner, ok := players[i]
				if !ok {
					winner = strconv.Itoa(i)
				}
				loser, ok := players[j]
				if !ok {
					loser = strconv.Itoa(j)
				}
				for n := 0; n < int(v); n++ {
					fmt.Fprintf(w, "\"%v\" -> \"%v\";\n", winner, loser)
				}
			}
		}
	}
	fmt.Fprintf(w, "}\n")
}

type Match struct {
	Winner string
	Loser  string
}

type Tournament []Match

func (t Tournament) Ids() map[string]int {
	ids := map[string]int{}
	id := 0
	for _, match := range t {
		if _, ok := ids[match.Winner]; !ok {
			ids[match.Winner] = id
			id++
		}
		if _, ok := ids[match.Loser]; !ok {
			ids[match.Loser] = id
			id++
		}
	}
	return ids
}

func (t Tournament) Players() map[int]string {
	ids := t.Ids()
	players := map[int]string{}
	for player, id := range ids {
		players[id] = player
	}
	return players
}

func (t Tournament) Matrix() *mat64.Dense {
	players := t.Ids()
	n := len(players)
	m := mat64.NewDense(n, n, nil)

	for _, match := range t {
		r := players[match.Winner]
		c := players[match.Loser]
		m.Set(r, c, m.At(r, c)+1)
	}
	return m
}
