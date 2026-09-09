package protocol

const (
	FacebookBaseURL  = "https://www.facebook.com"
	GraphQLBatchURL  = FacebookBaseURL + "/api/graphqlbatch/"
	GraphQLURL       = FacebookBaseURL + "/api/graphql/"
	MercuryUploadURL = FacebookBaseURL + "/ajax/mercury/upload.php"

	MessageRequestsDocID   = "3336396659757871"
	ThemeListDocID         = "24474714052117636"
	ThemeListFriendlyName  = "MWPThreadThemeQuery_AllThemesQuery"
	NoteCheckDocID         = "30899655739648624"
	NoteCheckFriendlyName  = "MWInboxTrayNoteCreationDialogQuery"
	NoteCreateDocID        = "24060573783603122"
	NoteCreateFriendlyName = "MWInboxTrayNoteCreationDialogCreationStepContentMutation"
	NoteDeleteDocID        = "9532619970198958"
	NoteDeleteFriendlyName = "useMWInboxTrayDeleteNoteMutation"
)

var RequiredSessionCookies = []string{"c_user", "xs"}
