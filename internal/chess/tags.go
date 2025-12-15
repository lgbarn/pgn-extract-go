package chess

// TagName represents the index of a predefined PGN tag.
type TagName int

const (
	AnnotatorTag TagName = iota
	BlackTag
	BlackEloTag
	BlackNATag
	BlackTitleTag
	BlackTypeTag
	BlackUSCFTag
	BoardTag
	DateTag
	ECOTag
	PseudoEloTag // Not a real PGN tag, used for matching either colour's rating
	EventTag
	EventDateTag
	EventSponsorTag
	FENTag
	PseudoFENPatternTag  // Not a real PGN tag, used for FEN-based pattern matching
	PseudoFENPatternITag // Inverted FEN pattern matching
	HashCodeTag
	LongECOTag
	MatchLabelTag    // For FENPattern matching indication
	MaterialMatchTag // For -z material pattern matching
	ModeTag
	NICTag
	OpeningTag
	PseudoPlayerTag // For matching either colour's player name
	PlyCountTag
	TotalPlyCountTag
	ResultTag
	RoundTag
	SectionTag
	SetupTag
	SiteTag
	StageTag
	SubVariationTag
	TerminationTag
	TimeTag
	TimeControlTag
	UTCDateTag
	UTCTimeTag
	VariantTag
	VariationTag
	WhiteTag
	WhiteEloTag
	WhiteNATag
	WhiteTitleTag
	WhiteTypeTag
	WhiteUSCFTag
	OriginalNumberOfTags // Sentinel, must be last
)

// TagNameStrings maps tag indices to their string representations.
var TagNameStrings = map[TagName]string{
	AnnotatorTag:         "Annotator",
	BlackTag:             "Black",
	BlackEloTag:          "BlackElo",
	BlackNATag:           "BlackNA",
	BlackTitleTag:        "BlackTitle",
	BlackTypeTag:         "BlackType",
	BlackUSCFTag:         "BlackUSCF",
	BoardTag:             "Board",
	DateTag:              "Date",
	ECOTag:               "ECO",
	PseudoEloTag:         "Elo",
	EventTag:             "Event",
	EventDateTag:         "EventDate",
	EventSponsorTag:      "EventSponsor",
	FENTag:               "FEN",
	PseudoFENPatternTag:  "FENPattern",
	PseudoFENPatternITag: "FENPatternI",
	HashCodeTag:          "HashCode",
	LongECOTag:           "LongECO",
	MatchLabelTag:        "MatchLabel",
	MaterialMatchTag:     "MaterialMatch",
	ModeTag:              "Mode",
	NICTag:               "NIC",
	OpeningTag:           "Opening",
	PseudoPlayerTag:      "Player",
	PlyCountTag:          "PlyCount",
	TotalPlyCountTag:     "TotalPlyCount",
	ResultTag:            "Result",
	RoundTag:             "Round",
	SectionTag:           "Section",
	SetupTag:             "SetUp",
	SiteTag:              "Site",
	StageTag:             "Stage",
	SubVariationTag:      "SubVariation",
	TerminationTag:       "Termination",
	TimeTag:              "Time",
	TimeControlTag:       "TimeControl",
	UTCDateTag:           "UTCDate",
	UTCTimeTag:           "UTCTime",
	VariantTag:           "Variant",
	VariationTag:         "Variation",
	WhiteTag:             "White",
	WhiteEloTag:          "WhiteElo",
	WhiteNATag:           "WhiteNA",
	WhiteTitleTag:        "WhiteTitle",
	WhiteTypeTag:         "WhiteType",
	WhiteUSCFTag:         "WhiteUSCF",
}

// StringToTagName maps tag strings to their indices.
var StringToTagName map[string]TagName

func init() {
	StringToTagName = make(map[string]TagName)
	for tag, name := range TagNameStrings {
		StringToTagName[name] = tag
	}
}

// SevenTagRoster contains the seven required PGN tags in order.
var SevenTagRoster = []string{
	"Event",
	"Site",
	"Date",
	"Round",
	"White",
	"Black",
	"Result",
}

// IsSevenTagRosterTag returns true if the tag is one of the seven required tags.
func IsSevenTagRosterTag(tag string) bool {
	for _, t := range SevenTagRoster {
		if t == tag {
			return true
		}
	}
	return false
}
