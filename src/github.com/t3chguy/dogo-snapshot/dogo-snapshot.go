package main

import (
	"flag"
	"fmt"
	"github.com/digitalocean/godo"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"sort"
	"strconv"
	"time"
)

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

type DateOrderedSnapshots []godo.Snapshot

func (ss DateOrderedSnapshots) Len() int {
	return len(ss)
}

func (ss DateOrderedSnapshots) Less(i, j int) bool {
	dateI, _ := time.Parse(time.RFC3339, ss[i].Created)
	dateJ, _ := time.Parse(time.RFC3339, ss[j].Created)
	return dateI.Before(dateJ)
}

func (ss DateOrderedSnapshots) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

func (ss DateOrderedSnapshots) filterPerDroplet(id int) (filtered DateOrderedSnapshots) {
	for _, snapshot := range ss {
		if snapshot.ResourceID == strconv.Itoa(id) {
			filtered = append(filtered, snapshot)
		}
	}
	return
}

func (ss DateOrderedSnapshots) filterName(name string) (filtered DateOrderedSnapshots) {
	for _, snapshot := range ss {
		if snapshot.Name == name {
			filtered = append(filtered, snapshot)
		}
	}
	return
}

func main() {
	var token, snapshotName string
	var maxSnapshots int

	flag.StringVar(&token, "token", "", "DigitalOcean Personal Access Token to use.")
	flag.StringVar(&snapshotName, "snapshot-name", "Automatic Snapshot", "The name of snapshots used by this program.")
	flag.IntVar(&maxSnapshots, "max-snapshots", 7, "Number of Snapshots to keep.")

	flag.Parse()

	if token == "" {
		panic("Need Token")
	}

	looseArgs := flag.Args()
	numLooseArgs := len(looseArgs)

	if numLooseArgs == 0 {
		panic("No Droplets Specified")
	}

	droplets := make([]int, 0, numLooseArgs)
	for _, dropletIdString := range looseArgs {
		num, _ := strconv.Atoi(dropletIdString)
		droplets = append(droplets, num)
	}

	oauthClient := oauth2.NewClient(context.Background(), &TokenSource{
		AccessToken: token,
	})
	client := godo.NewClient(oauthClient)

	ss, _, err := client.Snapshots.ListDroplet(context.Background(), nil)
	snapshots := DateOrderedSnapshots(ss).filterName(snapshotName)
	sort.Sort(snapshots)

	if err != nil {
		panic(err)
	}

	for _, dropletId := range droplets {
		fmt.Println("Cleaning Droplet", dropletId, "Snapshots")
		dropletSnapshots := snapshots.filterPerDroplet(dropletId)

		numSnapshots := len(dropletSnapshots)
		if numSnapshots >= 7 {
			for _, snapshot := range dropletSnapshots[0 : numSnapshots-(maxSnapshots-1)] {
				_, err := client.Snapshots.Delete(context.Background(), snapshot.ID)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}

	fmt.Println("")

	for _, dropletId := range droplets {
		fmt.Println("Snapshotting Droplet", dropletId)

		_, _, err := client.DropletActions.Snapshot(context.Background(), dropletId, snapshotName)

		if err != nil {
			fmt.Println(err)
		}
	}
}
