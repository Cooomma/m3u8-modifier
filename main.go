package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/grafov/m3u8"
	"github.com/urfave/cli/v2"
)

func main() {
	var input, output, edgeDomain, encryptedPath, appendedQueryParams string

	app := &cli.App{
		Name:  "m3u8 modifier",
		Usage: "modifier m3u8",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "input",
				Usage:       "Input Path",
				Value:       "",
				Aliases:     []string{"i"},
				Destination: &input,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "output",
				Usage:       "Output Path",
				Value:       "",
				Aliases:     []string{"o"},
				Destination: &output,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "edgeDomain",
				Usage:       "Edge Domain of CDN",
				Value:       "",
				Aliases:     []string{"e"},
				Destination: &edgeDomain,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "uri",
				Usage:       "Content URI",
				Value:       "",
				Aliases:     []string{"u"},
				Destination: &encryptedPath,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "query params",
				Usage:       "Segment URL Query Params",
				Value:       "",
				Aliases:     []string{"q"},
				Destination: &appendedQueryParams,
				Required:    false,
			},
		},
		Action: func(c *cli.Context) error {
			fmt.Println(input, output, edgeDomain, encryptedPath, appendedQueryParams)

			f, err := os.Open(input)
			if err != nil {
				panic(err)
			}
			defer f.Close()

			rawPlaylist, _, _ := m3u8.DecodeFrom(bufio.NewReader(f), true)
			rawMediaPlaylist := rawPlaylist.(*m3u8.MediaPlaylist)
			newMediaPlaylist, _ := createNewMediaPlaylist(rawMediaPlaylist, edgeDomain, encryptedPath, appendedQueryParams)

			if len(output) != 0 {
				fw, _ := os.Create(output)
				newMediaPlaylist.Encode().WriteTo(fw)
				defer fw.Close()
			} else {
				fmt.Println(newMediaPlaylist.Encode().String())
			}

			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func createNewMediaPlaylist(rawMediaPlaylist *m3u8.MediaPlaylist, edgeDomain, encryptedPath, appendedQueryParams string) (*m3u8.MediaPlaylist, error) {
	newMediaPlaylist, err := m3u8.NewMediaPlaylist(0, rawMediaPlaylist.Count())
	if err != nil {
		panic(err)
	}

	newMediaPlaylist.SetVersion(7)
	newMediaPlaylist.MediaType = m3u8.VOD
	newMediaPlaylist.TargetDuration = rawMediaPlaylist.TargetDuration
	initMp4URL, err := concatContentURL(edgeDomain, encryptedPath, "init.mp4")
	if err != nil {
		//FIXME: Report to upper layer
	}
	newMediaPlaylist.SetDefaultMap(
		initMp4URL,
		rawMediaPlaylist.Map.Limit,
		rawMediaPlaylist.Map.Offset,
	)

	for _, seg := range rawMediaPlaylist.Segments {
		if seg != nil {
			segmentURL, err := concatContentURL(edgeDomain, encryptedPath, seg.URI)
			if err != nil {
				//FIXME: Report to upper layer
			}
			err = newMediaPlaylist.Append(segmentURL, seg.Duration, seg.Title)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	newMediaPlaylist.Args = fmt.Sprintf("%s", appendedQueryParams)
	newMediaPlaylist.Closed = true
	return newMediaPlaylist, nil
}

func concatContentURL(domainURL, encryptedPath, objectName string) (string, error) {
	p, err := url.Parse(fmt.Sprintf("https://%s", strings.ReplaceAll(domainURL, "https://", "")))
	if err != nil {
		return "", err
	}
	p.Path = path.Join(p.Path, encryptedPath, objectName)
	return p.String(), nil
}
