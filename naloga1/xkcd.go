package main

import (
	"flag"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/laspp/PS-2024/vaje/naloga-1/koda/xkcd"
)

var wg sync.WaitGroup
var lock sync.Mutex
var mainDict = make(map[string]int)

type Beseda struct {
	key string
	val int
}

type voz struct {
	levo   *voz
	desno  *voz
	beseda Beseda
}

func (v *voz) Spucaj(meja int) *voz {
	tukaj := v
	for i := 0; i < meja && tukaj.desno != nil; i++ {
		tukaj = tukaj.desno
	}
	tukaj.desno = nil
	return v
}

func (v *voz) Dodaj(b Beseda) *voz {
	novo := &voz{beseda: b}
	//fmt.Println("Dodajam", b)
	if v == nil {
		//fmt.Println("v = nil, začenjam nov seznam.")
		v = novo
		return v
	}
	// beseda na začetek seznama
	if v.beseda.val <= novo.beseda.val {
		novo.desno = v
		return novo
	}

	tukaj := v
	for tukaj.desno != nil && tukaj.desno.beseda.val > novo.beseda.val {
		tukaj = tukaj.desno
	}
	novo.desno = tukaj.desno
	tukaj.desno = novo
	return v
}

func (v voz) PrintN(n int) {
	for i := 0; i < n && v.desno != nil; i++ {
		fmt.Println(v.beseda.key, ":", v.beseda.val)
		v = *v.desno
	}
}

func worker(id int, hopsize int, lastid int) {
	defer wg.Done()
	dict := make(map[string]int)
	re := regexp.MustCompile(`\b[a-zA-Z0-9]{4,}\b`)
	//fmt.Println("Running worker", id)
	// glavni loop
	for com := id; com <= lastid; com += hopsize {
		comic, err := xkcd.FetchComic(com)
		nit := ""
		if err == nil {
			if comic.Transcript == "" {
				nit = comic.Title + " " + comic.Tooltip
			} else {
				nit = comic.Title + " " + comic.Transcript
			}
		} else {
			fmt.Println("Worker", id, "encountered an error")
			return
		}
		words := re.FindAllString(nit, -1)
		for v := range words {
			words[v] = strings.ToLower(words[v])
			dict[words[v]]++
		}
	}
	// vse gre skupaj
	lock.Lock()
	for key, value := range dict {
		mainDict[key] += value
	}
	lock.Unlock()
}

func getLastId() int {
	comic, err := xkcd.FetchComic(0)
	if err == nil {
		return comic.Id
	} else {
		fmt.Println("Iskanje IDjev ni delalo.")
		return 0
	}
}

func main() {
	wPtr := flag.Int("w", 500, "Število delavcev")
	nPtr := flag.Int("n", 15, "Število izpisanih besed")
	flag.Parse()
	lastid := getLastId()
	hopsize := *wPtr
	for i := 0; i < hopsize; i++ {
		wg.Add(1)
		go worker(i, hopsize, lastid)
	}
	wg.Wait()

	// razvrščanje rezultatov
	var glava *voz
	i := 0
	for key, value := range mainDict {
		glava = glava.Dodaj(Beseda{key, value})
		i++
		// periodično zmanjšam seznam da ni potrebno sprehajanje čez celoten seznam.
		if i%73 == 0 {
			glava.Spucaj(*nPtr)
		}
	}
	// izpis rezultatov
	glava.PrintN(*nPtr)
}
