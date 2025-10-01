package repository

import "sync"

type User struct {
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	HashedPassword string `json:"password"`
	Age            int    `json:"age"`
	Gender         string `json:"gender"`
}

type UserStore struct {
	users  map[string]User
	nextID uint
	mu     sync.RWMutex
}

func NewUserStore(users map[string]User) *UserStore {
	return &UserStore{
		users:  users,
		nextID: 1,
		mu:     sync.RWMutex{},
	}
}

func (store *UserStore) AddUser(username, email, gender, hashedPassword string, age int) string {
	store.mu.Lock()
	user := User{
		ID:             store.nextID,
		Username:       username,
		Email:          email,
		HashedPassword: hashedPassword,
		Gender:         gender,
		Age:            age,
	}

	store.users[user.Email] = user
	store.nextID++
	store.mu.Unlock()

	return user.Email
}

func (store *UserStore) GetUserByEmail(email string) (User, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	user, ok := store.users[email]
	if !ok {
		return User{}, false
	}

	return user, ok
}
