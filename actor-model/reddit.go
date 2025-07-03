package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

// Messages
type RegisterUser struct {
	Username string
	Reply    chan string
}
type CreateSubreddit struct {
	UserID    int
	Subreddit string
	Reply     chan string
}
type JoinSubreddit struct {
	UserID    int
	Subreddit string
	Reply     chan string
}
type LeaveSubreddit struct {
	UserID    int
	Subreddit string
	Reply     chan string
}
type CreatePost struct {
	UserID    int
	Subreddit string
	Content   string
	Reply     chan string
}
type CommentOnPost struct {
	PostID  int
	UserID  int
	Content string
	Reply   chan string
}
type ReplyToComment struct {
	PostID    int
	CommentID int
	UserID    int
	Content   string
	Reply     chan string
}
type LikePost struct {
	PostID int
	UserID int
	Reply  chan string
}
type DislikePost struct {
	PostID int
	UserID int
	Reply  chan string
}
type SendMessage struct {
	SenderID   int
	ReceiverID int
	Content    string
	Reply      chan string
}
type ReplyToMessage struct {
	SenderID   int
	ReceiverID int
	Content    string
	Reply      chan string
}
type ViewInbox struct {
	UserID int
	Reply  chan string
}

// Actor States
type User struct {
	ID       int
	Username string
	Inbox    []string
}

type Subreddit struct {
	Name  string
	Users map[int]bool
	Posts []*Post
}

type Post struct {
	ID        int
	UserID    int
	Subreddit string
	Content   string
	Comments  []string
}

// EngineActor maintains the state of the application
type EngineActor struct {
	users      map[int]*User
	subreddits map[string]*Subreddit
	nextUserID int
	nextPostID int
	mu         sync.Mutex
}

// Helper method to get username by user ID
func (state *EngineActor) getUsername(userID int) string {
	if user, exists := state.users[userID]; exists {
		return user.Username
	}
	return fmt.Sprintf("User%d", userID)
}

// EngineActor Methods
func (state *EngineActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *RegisterUser:
		state.mu.Lock()
		state.nextUserID++
		user := &User{
			ID:       state.nextUserID,
			Username: msg.Username,
			Inbox:    []string{},
		}
		state.users[user.ID] = user
		state.mu.Unlock()
		msg.Reply <- fmt.Sprintf("\033[1;32mUser '%s' registered successfully with ID %d.\033[0m", msg.Username, user.ID)

	case *CreateSubreddit:
		state.mu.Lock()
		if _, exists := state.subreddits[msg.Subreddit]; exists {
			msg.Reply <- fmt.Sprintf("\033[1;32mSubreddit '%s' already exists.\033[0m", msg.Subreddit)
		} else {
			state.subreddits[msg.Subreddit] = &Subreddit{Name: msg.Subreddit, Users: map[int]bool{}}
			msg.Reply <- fmt.Sprintf("\033[1;32mSubreddit '%s' created successfully.\033[0m", msg.Subreddit)
		}
		state.mu.Unlock()

	case *JoinSubreddit:
		state.mu.Lock()
		if subreddit, exists := state.subreddits[msg.Subreddit]; exists {
			subreddit.Users[msg.UserID] = true
			msg.Reply <- fmt.Sprintf("\033[1;32mUser '%s' joined subreddit '%s'.\033[0m", state.getUsername(msg.UserID), msg.Subreddit)
		} else {
			msg.Reply <- fmt.Sprintf("\033[1;31mSubreddit '%s' does not exist.\033[0m", msg.Subreddit)
		}
		state.mu.Unlock()

	case *LeaveSubreddit:
		state.mu.Lock()
		if subreddit, exists := state.subreddits[msg.Subreddit]; exists {
			delete(subreddit.Users, msg.UserID)
			msg.Reply <- fmt.Sprintf("\033[1;32mUser '%s' left subreddit '%s'.\033[0m", state.getUsername(msg.UserID), msg.Subreddit)
		} else {
			msg.Reply <- fmt.Sprintf("\033[1;31mSubreddit '%s' does not exist.\033[0m", msg.Subreddit)
		}
		state.mu.Unlock()

	case *CreatePost:
		state.mu.Lock()
		state.nextPostID++
		if subreddit, exists := state.subreddits[msg.Subreddit]; exists {
			post := &Post{
				ID:        state.nextPostID,
				UserID:    msg.UserID,
				Subreddit: msg.Subreddit,
				Content:   msg.Content,
			}
			subreddit.Posts = append(subreddit.Posts, post)
			msg.Reply <- fmt.Sprintf("\033[1;32mUser '%s' posted in '%s': %s\033[0m", state.getUsername(msg.UserID), msg.Subreddit, msg.Content)
		} else {
			msg.Reply <- fmt.Sprintf("\033[1;31mSubreddit '%s' does not exist.\033[0m", msg.Subreddit)
		}
		state.mu.Unlock()

	case *CommentOnPost:
		state.mu.Lock()
		for _, subreddit := range state.subreddits {
			for _, post := range subreddit.Posts {
				if post.ID == msg.PostID {
					post.Comments = append(post.Comments, fmt.Sprintf("User '%s': %s", state.getUsername(msg.UserID), msg.Content))
					msg.Reply <- fmt.Sprintf("\033[1;32mUser '%s' commented on post %d: %s\033[0m", state.getUsername(msg.UserID), msg.PostID, msg.Content)
					state.mu.Unlock()
					return
				}
			}
		}
		state.mu.Unlock()
		msg.Reply <- fmt.Sprintf("\033[1;31mPost ID '%d' not found.\033[0m")

	case *ReplyToComment:
		state.mu.Lock()
		defer state.mu.Unlock()
		for _, subreddit := range state.subreddits {
			for _, post := range subreddit.Posts {
				if post.ID == msg.PostID {
					if msg.CommentID >= 1 && msg.CommentID <= len(post.Comments) {
						commentIndex := msg.CommentID - 1
						post.Comments[commentIndex] += fmt.Sprintf("\nReply by '%s': %s", state.users[msg.UserID].Username, msg.Content)
						msg.Reply <- fmt.Sprintf("\033[1;32mUser '%s' replied to comment %d on post %d: %s\033[0m", state.users[msg.UserID].Username, msg.CommentID, msg.PostID, msg.Content)
						return
					}
				}
			}
		}
		msg.Reply <- fmt.Sprintf("\033[1;31mComment or Post not found.\033[0m")

	case *LikePost:
		msg.Reply <- fmt.Sprintf("\033[1;32mUser '%s' liked post %d.\033[0m", state.getUsername(msg.UserID), msg.PostID)

	case *DislikePost:
		msg.Reply <- fmt.Sprintf("\033[1;32mUser '%s' disliked post %d.\033[0m", state.getUsername(msg.UserID), msg.PostID)

	case *SendMessage:
		state.mu.Lock()
		if receiver, exists := state.users[msg.ReceiverID]; exists {
			receiver.Inbox = append(receiver.Inbox, fmt.Sprintf("Message from '%s': %s", state.getUsername(msg.SenderID), msg.Content))
			msg.Reply <- fmt.Sprintf("\033[1;32mMessage sent to User '%s'.\033[0m", receiver.Username)
		} else {
			msg.Reply <- fmt.Sprintf("\033[1;31mUser with ID %d does not exist.\033[0m", msg.ReceiverID)
		}
		state.mu.Unlock()

	case *ReplyToMessage:
		state.mu.Lock()
		if receiver, exists := state.users[msg.ReceiverID]; exists {
			receiver.Inbox = append(receiver.Inbox, fmt.Sprintf("Reply from '%s': %s", state.getUsername(msg.SenderID), msg.Content))
			msg.Reply <- fmt.Sprintf("\033[1;32mReply sent to User '%s'.\033[0m", receiver.Username)
		} else {
			msg.Reply <- fmt.Sprintf("\033[1;31mUser with ID %d does not exist.\033[0m", msg.ReceiverID)
		}
		state.mu.Unlock()

	case *ViewInbox:
		state.mu.Lock()
		if user, exists := state.users[msg.UserID]; exists {
			if len(user.Inbox) == 0 {
				msg.Reply <- "\033[1;32mInbox is empty.\033[0m"
			} else {
				inboxContent := strings.Join(user.Inbox, "\n")
				msg.Reply <- fmt.Sprintf("\033[1;32mInbox:\n%s\033[0m", inboxContent)
			}
		} else {
			msg.Reply <- fmt.Sprintf("\033[1;31mUser with ID %d does not exist.\033[0m", msg.UserID)
		}
		state.mu.Unlock()
	}
}

// Main Function
func main() {
	engineActor := &EngineActor{
		users:      make(map[int]*User),
		subreddits: make(map[string]*Subreddit),
	}

	engineProps := actor.PropsFromProducer(func() actor.Actor { return engineActor })
	actorSystem := actor.NewActorSystem()
	engine := actorSystem.Root.Spawn(engineProps)

	fmt.Println("\033[1;36mWelcome to ProtoActor-based Social Media Platform!\033[0m")

	for {
		fmt.Println("\nChoose an option:")
		fmt.Println("1. Register User")
		fmt.Println("2. Create Subreddit")
		fmt.Println("3. Join Subreddit")
		fmt.Println("4. Create Post")
		fmt.Println("5. Comment on Post")
		fmt.Println("6. Reply to Comment")
		fmt.Println("7. Like Post")
		fmt.Println("8. Dislike Post")
		fmt.Println("9. Leave Subreddit")
		fmt.Println("10. Send Message")
		fmt.Println("11. Reply to Message")
		fmt.Println("12. View Inbox")
		fmt.Println("13. Exit")

		choice := readInput("Enter your choice: ")
		switch choice {
		case "1":
			username := readInput("Enter username: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &RegisterUser{Username: username, Reply: reply})
			fmt.Println(<-reply)
		case "2":
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			subreddit := readInput("Enter subreddit name: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &CreateSubreddit{UserID: userID, Subreddit: subreddit, Reply: reply})
			fmt.Println(<-reply)
		case "3":
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			subreddit := readInput("Enter subreddit name: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &JoinSubreddit{UserID: userID, Subreddit: subreddit, Reply: reply})
			fmt.Println(<-reply)
		case "4":
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			subreddit := readInput("Enter subreddit name: ")
			content := readInput("Enter post content: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &CreatePost{UserID: userID, Subreddit: subreddit, Content: content, Reply: reply})
			fmt.Println(<-reply)
		case "5":
			postID, _ := strconv.Atoi(readInput("Enter post ID: "))
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			content := readInput("Enter comment content: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &CommentOnPost{PostID: postID, UserID: userID, Content: content, Reply: reply})
			fmt.Println(<-reply)
		case "6":
			postID, _ := strconv.Atoi(readInput("Enter post ID: "))
			commentID, _ := strconv.Atoi(readInput("Enter comment ID to reply to: "))
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			content := readInput("Enter reply content: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &ReplyToComment{PostID: postID, CommentID: commentID, UserID: userID, Content: content, Reply: reply})
			fmt.Println(<-reply)
		case "7":
			postID, _ := strconv.Atoi(readInput("Enter post ID to like: "))
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			reply := make(chan string)
			actorSystem.Root.Send(engine, &LikePost{PostID: postID, UserID: userID, Reply: reply})
			fmt.Println(<-reply)
		case "8":
			postID, _ := strconv.Atoi(readInput("Enter post ID to dislike: "))
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			reply := make(chan string)
			actorSystem.Root.Send(engine, &DislikePost{PostID: postID, UserID: userID, Reply: reply})
			fmt.Println(<-reply)
		case "9":
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			subreddit := readInput("Enter subreddit name: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &LeaveSubreddit{UserID: userID, Subreddit: subreddit, Reply: reply})
			fmt.Println(<-reply)
		case "10":
			senderID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			receiverID, _ := strconv.Atoi(readInput("Enter receiver ID: "))
			content := readInput("Enter message content: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &SendMessage{SenderID: senderID, ReceiverID: receiverID, Content: content, Reply: reply})
			fmt.Println(<-reply)
		case "11":
			senderID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			receiverID, _ := strconv.Atoi(readInput("Enter receiver ID: "))
			content := readInput("Enter reply content: ")
			reply := make(chan string)
			actorSystem.Root.Send(engine, &ReplyToMessage{SenderID: senderID, ReceiverID: receiverID, Content: content, Reply: reply})
			fmt.Println(<-reply)
		case "12":
			userID, _ := strconv.Atoi(readInput("Enter your user ID: "))
			reply := make(chan string)
			actorSystem.Root.Send(engine, &ViewInbox{UserID: userID, Reply: reply})
			fmt.Println(<-reply)
		case "13":
			fmt.Println("\033[1;31mExiting... Goodbye!\033[0m")
			os.Exit(0)
		default:
			fmt.Println("\033[1;31mInvalid choice. Please try again.\033[0m")
		}
	}
}

// Utility function to read input
func readInput(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}
