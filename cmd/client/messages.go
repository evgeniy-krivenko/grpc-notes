package main

func getMessages() map[int]string {
	phrases := []string{
		"hello",
		"how are you?",
		"how does going?",
		"okey",
		"stay in touch",
		"nice to meet you",
		"good morning",
		"afternoon!",
		"hi, fellas!",
		"hello, people!",
	}

	messages := make(map[int]string, 10)

	for idx, phrase := range phrases {
		messages[idx] = phrase
	}

	return messages
}
