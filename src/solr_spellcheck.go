// solr_spellcheck scripts comes through synonyms dictionary
// and check words using spell checker aspell
package main

import (
	"github.com/trustmaster/go-aspell"
	"github.com/bitly/go-simplejson"
	"fmt"
	"os"
	"io/ioutil"
	"strings"
	"encoding/json"
	"encoding/gob"
	"strconv"
)

// symbolsMap is needed for reverse converting of
// language-specific symbols
var symbolsMap = map[string]map[string]string{
	"da": {"ae": "æ", "oe":"ø"},
	"no": {"ae": "æ", "oe":"ø"},
	"fi": {"ae": "æ", "oe":"ø", "th":"þ"},
	"nl": {"ij": "ĳ"},

}

// SpellChecker is global helper with spelling functions.
type SpellChecker struct {
	Locale string
	Speller aspell.Speller
}
var spellChecker SpellChecker

// replacements is map of processed words.
var replacements = make(map[string]string)

func main() {
	// Get a word from cmd line arguments.
	if len(os.Args) != 3 {
		fmt.Print("Usage: solr_spellcheck input_file locale\n")
		return
	}
	file := os.Args[1]
	local := os.Args[2]

	// We save already processed words in file.
	replacementsFile := local + ".gob"

	outputFile := local + ".processed.json"

	if _, err := os.Stat(replacementsFile); err == nil {
		// Open a RO file.
		decodeFile, err := os.Open(replacementsFile)
		if err != nil {
			panic(err)
		}
		// Create a decoder.
		decoder := gob.NewDecoder(decodeFile)

		// Read replacements.
		decoder.Decode(&replacements)
		decodeFile.Close()
	}

	// Create a file for IO.
	encodeFile, err := os.Create(replacementsFile)
	if err != nil {
		panic(err)
	}

	// Since this is a binary format large parts of it will be unreadable.
	encoder := gob.NewEncoder(encodeFile)

	// Write to the file in the end of script execution.
	defer func() {
		if err := encoder.Encode(replacements); err != nil {
			panic(err)
		}
		encodeFile.Close()
	}()

	// Initialize the speller.
	speller, err := aspell.NewSpeller(map[string]string{
		"lang": local,
	})
	if err != nil {
		fmt.Errorf("Error: %s", err.Error())
		return
	}
	defer speller.Delete()

	spellChecker.Locale = local
	spellChecker.Speller = speller

	// Read input file.
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return
	}

	// Decode file to *Json
	js, err := simplejson.NewJson(data)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// words - source map.
	words := js.Get("managedMap").MustMap()
	// result - processed words.
	result := make(map[string][]string)

	for base, s := range words {
		fmt.Println("Analyze: ", base, s)

		// Check base word.
		baseNew := ReplaceBySynonym(base)

		// Come through synonyms and process them.
		synonyms := js.Get("managedMap").Get(base).MustStringArray();
		for i, synonym := range synonyms {
			synonyms[i] = ReplaceBySynonym(synonym)
		}

		// Create a map with processed words.
		result[baseNew] = synonyms

		fmt.Println("-------------------------------------\n")
	}

	// Generate JSON output.
	jsonStringBin, _ := json.Marshal(result)
	jsonString := string(jsonStringBin)
	fmt.Println(jsonString)

	// Write output to file.
	err = ioutil.WriteFile(outputFile, jsonStringBin, 0644)
	if err != nil {
		panic(err)
	}
}

// SplitWords is used for splitting phrases.
// Words in phrases are concatenated by underscore.
func SplitWords(word string) []string {
	return strings.Split(word, "_");
}

// CheckWord returns suggestions of spelling if word has an error.
func CheckWord(word string) (status bool, suggestions []string) {
	speller := spellChecker.Speller
	newWord := word
	status = true

	// Do not process numbers.
	_, err := strconv.Atoi(word)
	if  err == nil {
		return
	}

	// Replace digraphs to relevant letters.
	for digraph, letter := range symbolsMap[spellChecker.Locale] {
		newWord = strings.Replace(newWord, digraph, letter, -1)
	}

	// Put corrected word to suggestions.
	if word != newWord {
		status = false
		suggestions = append(suggestions, newWord)
	}

	// Check spelling of corrected word and get spelling suggestions.
	if !speller.Check(newWord) {
		status = false
		suggestions = append(suggestions, speller.Suggest(newWord)...)
	}
	return status, suggestions
}

// GetSuggestionFromUser provides user interface, which allows selection
// of the correct for of the word.
func GetSuggestionFromUser(baseWord string, suggestions []string) (result int) {
	// If only one suggestion - accept it automatically.
	if (len(suggestions) == 1) {
		return 1
	}
	fmt.Printf("\nIncorrect word \"%s\". ", baseWord)
	if (len(suggestions) == 0) {
		fmt.Printf("No suggestions. \n")
		return 0
	}

	// Show list of the suggestions.
	fmt.Printf("Please select the suggestion:\n")
	fmt.Printf("0/Enter: Skip word. \n")
	for i, suggestionsWord := range suggestions {
		fmt.Printf("%d: %s \n", i + 1, suggestionsWord)
	}

	// Read value from user input.
	var i int
	_, err := fmt.Scanf("%d", &i)
	if (err != nil) {
		return 0
	}
	return i
}

// ReplaceBySynonym returns value of the processed replacement of the word.
func ReplaceBySynonym(word string) (replacement string) {
	words := SplitWords(word);
	for i, checkedWord := range words {
		_, ok := replacements[checkedWord];
		if ok {
			words[i] = replacements[checkedWord];
		} else {
			status, suggestions := CheckWord(checkedWord);
			if !status {
				index := GetSuggestionFromUser(checkedWord, suggestions)
				if (index != 0) {
					newWord := suggestions[index - 1]
					words[i] = newWord
					replacements[checkedWord] = newWord
				} else {
					replacements[checkedWord] = checkedWord
				}
			}
		}
	}
	return strings.Join(words, "_")
}
