package main

import "time"

type App struct {
	UserName     string
	Password     string
	ErrorMsg     string
	Session      BSkySession
	Timeline     Timeline
	ShowImages   bool
	LoginPending bool
	LoopCancel   chan struct{}

	// RevealAnchorID is the view ID of the post that was at the top of
	// the timeline before a refresh prepended new posts. Consumed by
	// revealAmend on the next layout pass: the old content is anchored
	// in place, then the view eases up to reveal the new posts.
	RevealAnchorID string
}

type Timeline struct {
	Posts []Post
}

type Post struct {
	ID                     string
	Author                 string
	Verified               bool
	CreatedAt              time.Time
	Text                   string
	LinkURI                string
	LinkTitle              string
	ImagePath              string
	ImageAlt               string
	ImageWidth             float32
	ImageHeight            float32
	RepostBy               string
	Replies                int
	Reposts                int
	Likes                  int
	BSkyLinkURI            string
	QuotePostAuthor        string
	QuotePostCreatedAt     time.Time
	QuotePostText          string
	QuotePostLinkTitle     string
	QuotePostLinkURI       string
	FormattedText          string
	FormattedRepostBy      string
	FormattedTimeAuthor    string
	FormattedQuoteText     string
	FormattedQuoteTimeAuth string
}

type BSkySession struct {
	Handle         string `json:"handle" toml:"handle"`
	Email          string `json:"email" toml:"email"`
	EmailConfirmed bool   `json:"emailConfirmed" toml:"emailConfirmed"`
	AccessJWT      string `json:"accessJwt" toml:"accessJwt"`
	RefreshJWT     string `json:"refreshJwt" toml:"refreshJwt"`
}

type bSkyCreateSessionRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type refreshSessionResponse struct {
	AccessJWT  string `json:"accessJwt"`
	RefreshJWT string `json:"refreshJwt"`
	Active     bool   `json:"active"`
}

type bSkyTimeline struct {
	Posts []bSkyPost `json:"feed"`
}

type bSkyPost struct {
	Post struct {
		URI    string     `json:"uri"`
		Author bSkyAuthor `json:"author"`
		Record struct {
			Type      string      `json:"$type"`
			Text      string      `json:"text"`
			CreatedAt string      `json:"createdAt"`
			Embed     bSkyEmbed   `json:"embed"`
			Reply     bSkyReply   `json:"reply"`
			Facets    []bSkyFacet `json:"facets"`
		} `json:"record"`
		Embed struct {
			Type      string `json:"$type"`
			CID       string `json:"cid"`
			Thumbnail string `json:"thumbnail"`
			Record    struct {
				Type   string     `json:"$type"`
				Author bSkyAuthor `json:"author"`
				Value  bSkyValue  `json:"value"`
			} `json:"record"`
		} `json:"embed"`
		Replies int `json:"replyCount"`
		Likes   int `json:"likeCount"`
		Reposts int `json:"repostCount"`
		Quotes  int `json:"quoteCount"`
	} `json:"post"`
	Reason struct {
		Type string     `json:"$type"`
		By   bSkyAuthor `json:"by"`
	} `json:"reason"`
}

type bSkyAuthor struct {
	DID          string `json:"did"`
	Handle       string `json:"handle"`
	DisplayName  string `json:"displayName"`
	Verification struct {
		VerifiedStatus string `json:"verifiedStatus"`
	} `json:"verification"`
}

type bSkyEmbed struct {
	Type     string           `json:"$type"`
	Images   []bSkyImageLink  `json:"images"`
	Media    bSkyMedia        `json:"media"`
	External bSkyExternalLink `json:"external"`
}

type bSkyImageLink struct {
	Alt   string `json:"alt"`
	Image struct {
		Type string `json:"$type"`
		Ref  struct {
			Link string `json:"$link"`
		} `json:"ref"`
	} `json:"image"`
}

type bSkyMedia struct {
	Type   string          `json:"$type"`
	Images []bSkyImageLink `json:"images"`
}

type bSkyExternalLink struct {
	Title string `json:"title"`
	URI   string `json:"uri"`
}

type bSkyValue struct {
	Type      string      `json:"$type"`
	CreatedAt string      `json:"createdAt"`
	Text      string      `json:"text"`
	Embed     bSkyEmbed   `json:"embed"`
	Facets    []bSkyFacet `json:"facets"`
}

type bSkyFacet struct {
	Features []struct {
		Type string `json:"$type"`
		URI  string `json:"uri"`
	} `json:"features"`
	Index struct {
		ByteStart int `json:"byteStart"`
		ByteEnd   int `json:"byteEnd"`
	} `json:"index"`
}

type bSkyReply struct {
	Parent struct {
		CID string `json:"cid"`
	} `json:"parent"`
	Root struct {
		CID string `json:"cid"`
	} `json:"root"`
}

type imageSource struct {
	CID       string
	URL       string
	Alt       string
	AuthorDID string
}
