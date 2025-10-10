package fork

import (
	"project/domain"
	"sync"
)

type PostsStore struct {
	posts []domain.Post
	mu    sync.RWMutex
}

func NewPostStore(posts []domain.Post) *PostsStore {
	return &PostsStore{
		posts: posts,
		mu:    sync.RWMutex{},
	}
}

func (store *PostsStore) PostsPaginatedList(page, limit int) ([]domain.Post, int) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	start := (page - 1) * limit
	end := start + limit
	length := len(store.posts)
	pagesCount := (length + limit - 1) / limit

	if start > length {
		start = length
	}

	if end > length {
		end = length
	}

	sliced := store.posts[start:end]
	copySlice := make([]domain.Post, len(sliced))
	copy(copySlice, sliced)

	return copySlice, pagesCount
}

var ForkPosts = []domain.Post{
	{1, "С днём рождения, Boulevard Depo!\nТалантливому рэп-артисту, стоявшему у истоков творческого объединения YungRussia и продолжающему развивать успешную сольную карьеру, сегодня исполнилось 34 года.\n#ffmbirthdays", 12, 12, 12, "Fast Food Music", "/asserts/groupim.jpg", []string{"/asserts/postImage.jpg"}},
	{2, "В интернет слили 17 фристайлов Kanye West, записанных для альбома Travis Scott «UTOPIA»\n\nНи для кого не секрет, что при работе над своими треками Трэвис собирает большую команду, помогающую ему в тех или иных аспектах.\n\nТак, последний на данный момент сольный альбом артиста «UTOPIA» сильно напоминает творчество Kanye West, в частности — «Yeezus». Это неслучайно, ведь Йе выступил одним из главных вдохновителей Трэва при создании релиза и даже записал для него 17 референсов, которые сегодня и были слиты в интернет.", 35, 24, 1, "Fast Food Music", "/asserts/groupim.jpg", []string{"/asserts/8.jpg"}},
	{3, "Nacho Picasso & Televangel — «Séance Musique»\nGenre: Hip-Hop\nLabel: «Last Epoch Recordings»\nApple Music: vk.cc/cPS5do / Spotify: vk.cc/cPS6qG\n\nВидный деятель андерграундной сцены Сиэтла Nacho Picasso в строю более четырнадцати лет. Он покорил слушателей рядом достойных «сольников», на которых сумел найти грамотный баланс между психоделической эстетикой клауд-рэпа и настырным трэп-звучанием, сопровождающим его мрачные тексты, также нередко построенные на вульгарном и чёрном юморе.", 24, 16, 1, "Fast Food Music", "/asserts/groupim.jpg", []string{"/asserts/2.jpg"}},
	{4, "Продажи альбомов (19 – 26 августа)\n\nКарди ожидаемо заняла лидирующую позицию чарта, а её альбом стал платиновым в первую неделю за счёт того, что артистка и её лейбл поместили хитовые синглы 5-летней давности «WAP» и «Up» в треклист.\n\nМикстейп участника Opium же оказался гораздо ниже и показал довольно скромные результаты.", 20, 4, 2, "Fast Food Music", "/asserts/groupim.jpg", []string{"/asserts/3.jpg"}},
	{5, "Yung Lean & Bladee — «Evil World»\n\nСудя по всему, близкие друзья и давние коллеги в лице Yung Lean и Bladee ведут работу над новым совместным альбомом. На прошлой неделе шведские артисты представили запоминающуюся работу «Inferno», а теперь вернулись с ещё одной композицией.\n\n«Evil World» продолжает развивать многослойное звучание на стыке клауд-рэпа, рейджа и электроники, заданное предыдущим синглом.", 102, 6, 15, "Fast Food Music", "/asserts/groupim.jpg", []string{"/asserts/4.jpg"}},
	{6, "Yung Lean & Bladee — «Evil World»\n\nСудя по всему, близкие друзья и давние коллеги в лице Yung Lean и Bladee ведут работу над новым совместным альбомом. На прошлой неделе шведские артисты представили запоминающуюся работу «Inferno», а теперь вернулись с ещё одной композицией.\n\n«Evil World» продолжает развивать многослойное звучание на стыке клауд-рэпа, рейджа и электроники, заданное предыдущим синглом.", 97, 72, 1, "Fast Food Music", "/asserts/groupim.jpg", []string{"/asserts/5.jpg"}},
	{7, "Peezy — «STILL GHETTTO»\nGenre: Hip-Hop\nLabel: «#Boyz Entertainment», «EMPIRE»\nApple Music: vk.cc/cPQlBc / Spotify: vk.cc/cPRKKE\n\nУроженец Детройта Peezy по праву считается крёстным отцом современной мичиганской сцены, породившей множество трэп-артистов. В своём творчестве он часто использует элементы сторителлинга, погружая слушателей в суровые реалии родных районов и рассказывая истории местных жителей.\n\nВ 2023-м Peezy выпустил один из своих самых крупных альбомов «GHETTO». Сегодня же он возвращается с его идейным продолжением, которое также отражает непоколебимую связь автора с улицами, сформировавшими его личность.\n\nСписок гостей достоин отдельного упоминания: в создании релиза приняли участие Rick Ross, 2 Chainz, Big Sean, Larry June, French Montana, G Herbo, 42 Dugg, Jeremih, Babyface Ray, Icewear Vezzo и другие.", 75, 29, 3, "Fast Food Music", "/asserts/groupim.jpg", []string{"/asserts/6.jpg"}},
	{8, "Marino Infantry — «M:4»\nGenre: Hip-Hop\nLabel: «Marino Infantry Records»\nApple Music: vk.cc/cPRJyz / Spotify: vk.cc/cPRJzf\n\nA$AP Ant по-прежнему остаётся самым продуктивным артистом, связанным с творческим объединением A$AP Mob. В этом году он уже успел отметиться сольным альбомом «Addie Pitino 2», однако это далеко не весь материал, который он приготовил для поклонников своего творчества.\n\nТеперь исполнитель решил обратить внимание аудитории на других участников своего коллектива Marino Infantry путём выпуска компиляции «M:4». Впрочем, это скорее напоминает очередной микстейп Энта — просто с большим количеством гостевых участий.", 46, 24, 4, "Fast Food Music", "/asserts/groupim.jpg", []string{"/asserts/7.jpg"}},
}
