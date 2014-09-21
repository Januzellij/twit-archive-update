package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Januzellij/anaconda"
	"github.com/codegangsta/cli"
)

// why is this global?
// - It is mutated only once, at the start of main()
// - It is used in most functions
var archiveFolder string

type tweetSlice []anaconda.Tweet

func (t tweetSlice) Len() int      { return len(t) }
func (t tweetSlice) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t tweetSlice) Less(i, j int) bool {
	firstTime, err := t[i].CreatedAtTime()
	if err != nil {
		log.Fatal(err)
	}
	secondTime, err := t[j].CreatedAtTime()
	if err != nil {
		log.Fatal(err)
	}
	return firstTime.After(secondTime) // sorts with the most recent time at index 0
}

type tweetIndex struct {
	FileName   string  `json:"file_name"`
	Year       float64 `json:"year"`
	VarName    string  `json:"var_name"`
	TweetCount float64 `json:"tweet_count"`
	Month      float64 `json:"month"`
}

type fileDate struct {
	year, month                 int
	filename, tweetsMonthString string
}

// makeFileDate creates a fileDate with a proper filename
func makeFileDate(year, month int) fileDate {
	var filename string
	if month < 10 {
		filename = archiveFolder + fmt.Sprintf("/data/js/tweets/%d_0%d.js", year, month)
	} else {
		filename = archiveFolder + fmt.Sprintf("/data/js/tweets/%d_%d.js", year, month)
	}
	// ends up as "tweets_year_month"
	tweetsMonthString := "tweets_" + strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(filename, archiveFolder), "/data/js/tweets/"), ".js")
	return fileDate{year, month, filename, tweetsMonthString}
}

// fileDate.tweetIndex converts a fileDate to a tweetIndex
func (f fileDate) tweetIndex(t tweetSlice) tweetIndex {
	archiveLocalFilename := strings.TrimPrefix(f.filename, archiveFolder) // chops off the tweets folder path from the filename
	return tweetIndex{archiveLocalFilename,
		float64(f.year),
		f.tweetsMonthString,
		float64(len(t)),
		float64(f.month),
	}
}

type tweetIndexSlice []tweetIndex

func (t tweetIndexSlice) Len() int      { return len(t) }
func (t tweetIndexSlice) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t tweetIndexSlice) Less(i, j int) bool {
	if t[i].Year == t[j].Year {
		return t[i].Month > t[j].Month
	} else {
		return t[i].Year > t[j].Year
	}
	// sort with most recent tweetIndex at index 0
}

// expandPathArg expands out ~'s to the users home directory
func expandPathArg(path string) string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(path, "~", usr.HomeDir, -1)
}

// parseArchiveCreated retrieves the time the archive was last modified
func parseArchiveCreated() (archiveCreated time.Time) {
	details, err := ioutil.ReadFile(archiveFolder + "/data/js/payload_details.js")
	if err != nil {
		log.Fatal(err)
	}
	detailsJSON := details[22:]
	// nasty hack: this particular .js file will always have 21 bytes before the actual JSON begins, so slicing off those bytes will always give the JSON
	// why use it? The alternative is converting the whole file to a string, splitting it into a 2 element slice,
	// taking the last element and converting that into a byte slice
	// Although this file is very small, and we could do that, I do the same thing
	// to some very big files below (because of performance), and I figured it might as well be consistent
	var detailsMap map[string]interface{}
	err = json.Unmarshal(detailsJSON, &detailsMap)
	if err != nil {
		log.Fatal(err)
	}
	archiveCreated, err = time.Parse("2006-01-02 15:04:05 -0700", detailsMap["created_at"].(string))
	if err != nil {
		log.Fatal(err)
	}
	return
}

// writeJSONToFile removes a file, creates it again, and writes the json to it
func writeJSONToFile(filename string, jsonData interface{}, prefix []byte) {
	newJSON, err := json.Marshal(jsonData)
	if err != nil {
		log.Fatal(err)
	}
	if err = os.Remove(filename); err != nil {
		log.Fatal(err)
	}
	newFile, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer newFile.Close()
	if _, err = newFile.Write(prefix); err != nil {
		log.Fatal(err)
	}
	if _, err = newFile.Write(newJSON); err != nil {
		log.Fatal(err)
	}
}

// updates metadata in payload_details.js and tweet_index.js
func updateMetadata(number int, fileMap map[fileDate]tweetSlice) {
	payloadFile := archiveFolder + "/data/js/payload_details.js"
	details, err := ioutil.ReadFile(payloadFile)
	if err != nil {
		log.Fatal(err)
	}
	detailsBeginning := details[:22]
	detailsJSON := details[22:]
	var detailsMap map[string]interface{}
	err = json.Unmarshal(detailsJSON, &detailsMap)
	if err != nil {
		log.Fatal(err)
	}
	detailsMap["tweets"] = detailsMap["tweets"].(float64) + float64(number)         // updates the number of total tweets
	detailsMap["created_at"] = time.Now().UTC().Format("2006-01-02 15:04:05 -0700") // updates the "last updated" time to now
	writeJSONToFile(payloadFile, detailsMap, detailsBeginning)

	indexFile := archiveFolder + "/data/js/tweet_index.js"
	index, err := ioutil.ReadFile(indexFile)
	if err != nil {
		log.Fatal(err)
	}
	indexBeginning := index[:19]
	indexJSON := index[19:]
	var indices tweetIndexSlice
	err = json.Unmarshal(indexJSON, &indices)
	if err != nil {
		log.Fatal(err)
	}
	newIndices := make(tweetIndexSlice, len(fileMap))
	i := 0
	for k, v := range fileMap {
		newIndices[i] = k.tweetIndex(v)
		i++
	}
	sort.Sort(newIndices)
	indices = append(newIndices, indices...)
	// deletes any duplcate indices
IndicesLoop: // TODO: is the label needed?
	for i, v := range indices {
		if i == len(indices)-1 {
			break IndicesLoop // this is the last index, obtaining nextIndice would result in a runtime error
		}
		nextIndice := indices[i+1]
		if v.Year == nextIndice.Year && v.Month == nextIndice.Month {
			indices = append(indices[:i+1], indices[i+2:]...) // delete the duplicate at i+1 (the old index)
			break IndicesLoop
		}
	}
	writeJSONToFile(indexFile, indices, indexBeginning)
}

// pretty self explanatory
func createAPI() *anaconda.TwitterApi {
	apiKey := os.Getenv("TU_KEY")
	apiSecret := os.Getenv("TU_SECRET")
	accessToken := os.Getenv("TU_TOKEN")
	accessTokenSecret := os.Getenv("TU_TOKEN_SECRET")
	if apiKey == "" || apiSecret == "" || accessToken == "" || accessTokenSecret == "" {
		log.Fatal("Could not find required environment variables\nNeeded:\nTU_KEY\nTU_SECRET\nTU_TOKEN\nTU_TOKEN_SECRET\n")
	}
	anaconda.SetConsumerKey(apiKey)
	anaconda.SetConsumerSecret(apiSecret)
	return anaconda.NewTwitterApi(accessToken, accessTokenSecret)
}

// fetchNewTweets fetches all tweets from when the archive was last modified to now
func fetchNewTweets(archiveCreated time.Time) (tweets tweetSlice) {
	api := createAPI()
	var lowestPreviousID int64
	var earliestTweetTime time.Time
	// when fetchNewTweets returns, earliestTweetTime will actually be eariler than archiveCreated, since it is in charge of stopping the loop
	firstTweet, firstTimeline := true, true
TweetTimeLoop:
	for earliestTweetTime.After(archiveCreated) || earliestTweetTime.IsZero() {
		params := url.Values{"count": []string{"20"}}
		if !firstTimeline {
			params["max_id"] = []string{strconv.FormatInt(lowestPreviousID-1, 10)} // see: https://dev.twitter.com/docs/working-with-timelines
		} else {
			firstTimeline = false
		}
		timeline, err := api.GetUserTimeline(params)
		if err != nil {
			log.Fatal(err)
		}
		for _, tweet := range timeline {
			if firstTweet || tweet.Id < lowestPreviousID {
				lowestPreviousID = tweet.Id
			}
			tweetTime, err := tweet.CreatedAtTime()
			if err != nil {
				log.Fatal(err)
			}
			if firstTweet || tweetTime.Before(earliestTweetTime) {
				earliestTweetTime = tweetTime
			}
			if firstTweet {
				firstTweet = false
			}
			if earliestTweetTime.Before(archiveCreated) {
				break TweetTimeLoop
			}
			tweets = append(tweets, tweet)
		}
	}
	return
}

// genFileMap sorts a slice of tweets according to which file they go in (according to which month they were tweeted)
func genFileMap(tweets tweetSlice) (fileMap map[fileDate]tweetSlice) {
	sort.Sort(tweets)
	fileMap = make(map[fileDate]tweetSlice)
	monthStart := 0
	var currentMonth int
	for i, tweet := range tweets {
		tweetTime, err := tweet.CreatedAtTime()
		if err != nil {
			log.Fatal(err)
		}
		tweetMonth := int(tweetTime.Month())
		if i == 0 {
			currentMonth = tweetMonth
		}
		tweetDate := makeFileDate(tweetTime.Year(), tweetMonth)
		if _, ok := fileMap[tweetDate]; !ok {
			if tweetMonth != currentMonth {
				monthBefore := makeFileDate(tweetTime.Year(), tweetMonth-1)
				fileMap[monthBefore] = tweets[monthStart:i]
				monthStart = i
				currentMonth = tweetMonth
			} else if i == len(tweets)-1 {
				fileMap[tweetDate] = tweets
			}
		}
	}
	return
}

// writeFileMap writes a fileMap to the requisite files
func writeFileMap(fileMap map[fileDate]tweetSlice) {
	for k, v := range fileMap {
		file, err := ioutil.ReadFile(k.filename)
		if err != nil {
			newFile, err := os.Create(k.filename)
			if err != nil {
				log.Fatal(err)
			}
			defer newFile.Close()
			newArchiveTweets := anaconda.ConvertToArchive([]anaconda.Tweet(v))
			newJSON, err := json.Marshal(newArchiveTweets)
			if err != nil {
				log.Fatal(err)
			}
			prefix := "Grailbird.data." + k.tweetsMonthString + " = \n "
			if _, err = newFile.WriteString(prefix); err != nil {
				log.Fatal(err)
			}
			if _, err = newFile.Write(newJSON); err != nil {
				log.Fatal(err)
			}
		} else {
			// append to the existing month file
			var JSONTweets []anaconda.ArchiveTweet
			fileBeginning := file[:33]
			fileJSON := file[33:]
			err = json.Unmarshal(fileJSON, &JSONTweets)
			if err != nil {
				log.Fatal(err)
			}
			newArchiveTweets := anaconda.ConvertToArchive([]anaconda.Tweet(v))
			JSONTweets = append(newArchiveTweets, JSONTweets...)
			writeJSONToFile(k.filename, JSONTweets, fileBeginning)
		}
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "twit-archive-update"
	app.Usage = "updates a local Twitter archive with a users latest tweets"
	app.Action = func(c *cli.Context) {
		archiveFolder = expandPathArg(c.Args().First())
		archiveCreated := parseArchiveCreated()
		tweets := fetchNewTweets(archiveCreated)
		fileMap := genFileMap(tweets)
		// TODO:
		// figure out why the JSON files get so much bigger (not enough omitempty?)
		// get URL's to show up
		// - HTML:
		//<a class="link" href="http://t.co/krif9rR62a" target="_blank" title="http://twitter.com/Januzellij/status/504247052667084800/photo/1">
		// pic.twitter.com/krif9rR62a</a>
		// update user_details.js with any new user details
		writeFileMap(fileMap)
		//updateMetadata(len(tweets), fileMap)
	}
	app.Run(os.Args)
}
