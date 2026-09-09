package protocol

const (
	FacebookBaseURL  = "https://www.facebook.com"
	GraphQLBatchURL  = FacebookBaseURL + "/api/graphqlbatch/"
	GraphQLURL       = FacebookBaseURL + "/api/graphql/"
	MercuryUploadURL = FacebookBaseURL + "/ajax/mercury/upload.php"
)

var RequiredSessionCookies = []string{"c_user", "xs"}
