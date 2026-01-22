package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"wfts/configs"
	"wfts/internal/model"
	"wfts/internal/repository"
	"wfts/internal/services/tui"
	"wfts/internal/services/wfts/offline/indexer"
	"wfts/internal/services/wfts/online/searcher"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var (
		configFile = flag.String("config", "configs/app_config.json", "Path to configuration file")
		indexFlag = flag.Bool("i", false, "disable indexing")
		interfaceFlag = flag.Bool("gui", false, "use terminal UI")
	)
	flag.Parse()

	cfg, err := configs.UploadLocalConfiguration(*configFile)
	if err != nil {
		panic(err)
	}

	if *interfaceFlag {
		initGUI(cfg, *indexFlag)
		return
	}
	
	out := os.Stdout
	if cfg.InfoLogPath != "-" {
		out, err = os.Create(cfg.InfoLogPath)
		if err != nil {
			panic(err)
		}
	}
	defer out.Close()

	ir, err := repository.NewIndexRepository(cfg.IndexPath, out, cfg.ChunkSize)
	if err != nil {
		panic(err)
	}
	defer ir.DB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		fmt.Println("\nShutting down...")
		cancel()
		//os.Exit(1)
	}()

	i := indexer.NewIndexer(ir, out, cfg)
	if !*indexFlag {
		if err := i.Index(cfg, ctx); err != nil {
			panic(err)
		}
	}

	count, err := ir.GetDocumentsCount()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Index built with %d documents. Enter search queries (q to exit):\n", count)

	s := searcher.NewSearcher(out, i, ir)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ")
		query, _ := reader.ReadString('\n')
		query = strings.TrimSpace(query)
		if query == "q" {
			return
		}
		t := time.Now()
		Present(s.Search(query, 100))
		fmt.Printf("--Search time: %v--\n", time.Since(t))
	}
}

func Present(docs []*model.Document) {
	if len(docs) == 0 {
		fmt.Println("No results found.")
		return
	}
	
	fmt.Printf("Found %d results:\n", len(docs))
	for i, doc := range docs {
		fmt.Printf("%d. URL: %s\n\n", 
			i+1, doc.URL)
	}
}

func initGUI(cfg *configs.ConfigData, indexF bool) {
	lc := tui.NewLogChannel(cfg.LogChannelSize)
	ir, err := repository.NewIndexRepository(cfg.IndexPath, lc, cfg.ChunkSize)
	if err != nil {
		panic(err)
	}
	defer ir.DB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan struct{}, 1)
	go func() {
		<-c
		cancel()
		//os.Exit(1)
	}()

	i := indexer.NewIndexer(ir, lc, cfg)
	if !indexF {
		go func() {
			if err := i.Index(cfg, ctx); err != nil {
				panic(err)
			}
		}()
	}

	model := tui.InitModel(lc, cfg.TUIBorderColor, ir.GetDocumentsCount, searcher.NewSearcher(lc, i, ir).Search, c)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		panic(err)
	}
}