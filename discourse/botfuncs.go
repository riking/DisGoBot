package discourse

//import "github.com/riking/discourse/discourse"
import "fmt"

type SeeEveryPostCallback func(S_Post) ()

const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 1)

func SeeEveryPost(bot *DiscourseSite, highestSeen *int, callback SeeEveryPostCallback, onlyBelow int) {
	var posts ResponseLatestPosts
	var request string
	var myHighest int = 0

	lowestId := MaxInt
	if onlyBelow > 0 {
		lowestId = onlyBelow
	}

	for lowestId > *highestSeen {
		if request == "" && onlyBelow <= 0 {
			request = "/posts.json" // first loop
		} else {
			request = fmt.Sprintf("/posts.json?before=%d", lowestId)
		}

		err := bot.DGetJsonTyped(request, &posts)
		if err != nil {
			fmt.Println(err)
			return
		}


		for _, post := range posts.Latest_posts {
			if post.Id < lowestId && post.Id > *highestSeen {
				//				fmt.Println(post.Id)
				callback(post)
			}
			if post.Id > myHighest {
				myHighest = post.Id
			}
		}
		if lowestId == MaxInt {
			lowestId = posts.Latest_posts[0].Id // not optimal
		} else {
			lowestId = lowestId-50
		}
	}
	*highestSeen = myHighest
}
