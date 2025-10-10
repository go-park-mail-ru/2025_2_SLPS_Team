package fork

import (
	"project/domain"
	"sync"
)

type UserStore struct {
	users  map[string]domain.User
	nextID uint
	mu     sync.RWMutex
}

func NewUserStore(users map[string]domain.User) *UserStore {
	return &UserStore{
		users:  users,
		nextID: 1,
		mu:     sync.RWMutex{},
	}
}

func (store *UserStore) AddUser(firstname, lastname, email, gender, hashedPassword string, age int) string {
	store.mu.Lock()
	user := domain.User{
		ID:        store.nextID,
		FirstName: firstname,
		LastName:  lastname,
		Email:     email,
		Password:  hashedPassword,
		Gender:    gender,
		Age:       age,
	}

	store.users[user.Email] = user
	store.nextID++
	store.mu.Unlock()

	return user.Email
}

func (store *UserStore) GetUserByEmail(email string) (domain.User, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	user, ok := store.users[email]
	if !ok {
		return domain.User{}, false
	}

	return user, ok
}
