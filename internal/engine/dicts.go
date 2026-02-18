package engine

var (
	LastNames  = []string{"김", "이", "박", "최", "정", "강", "조", "윤", "장", "임", "한", "오", "서", "신", "권", "황", "안", "송", "류", "전"}
	FirstNames = []string{"민준", "서준", "도윤", "예준", "시우", "하준", "지호", "주원", "지우", "준우", "서연", "서윤", "지우", "서현", "하은", "민서", "지유", "윤서", "채원"}
	Cities     = []string{"서울", "부산", "대구", "인천", "광주", "대전", "울산", "수원", "성남", "고양", "용인", "부천", "안산", "청주", "전주", "천안", "남양주", "화성", "안양", "김해"}
	Districts  = []string{"강남구", "서초구", "송파구", "종로구", "마포구", "영등포구", "관악구", "동작구", "강동구", "노원구", "은평구", "서대문구", "성북구", "동대문구", "중랑구"}
	Streets    = []string{"테헤란로", "강남대로", "송파대로", "올림픽로", "한강대로", "세종대로", "을지로", "퇴계로", "충무로", "종로", "신촌로", "양화로", "경인로", "시흥대로", "남부순환로"}
)

// 영-한 단어 사전 (번역 시뮬레이션용) - 약 200개 단어
var EngToKorMap = map[string]string{
	// 장르 & 명사
	"Action": "액션", "Adventure": "모험", "Animation": "애니메이션",
	"Children": "아동", "Classics": "고전", "Comedy": "코미디",
	"Documentary": "다큐멘터리", "Drama": "드라마", "Family": "가족",
	"Foreign": "외국", "Games": "게임", "Horror": "공포",
	"Music": "음악", "New": "새로운", "Sci-Fi": "SF",
	"Sports": "스포츠", "Travel": "여행",
	"Movie": "영화", "Film": "작품", "Story": "이야기",
	"Life": "인생", "Time": "시간", "World": "세계",
	"Man": "남자", "Woman": "여자", "Hero": "영웅",
	"Friend": "친구", "Enemy": "적", "Love": "사랑",
	"War": "전쟁", "Peace": "평화", "Hope": "희망",
	"Dream": "꿈", "Fate": "운명", "Memory": "기억",
	"Truth": "진실", "Lie": "거짓", "Secret": "비밀",
	"Legend": "전설", "Mystery": "미스터리", "Battle": "전투",
	"Future": "미래", "Past": "과거", "Present": "현재",
	"Journey": "여정", "Destiny": "숙명", "Glory": "영광",
	"Honor": "명예", "Freedom": "자유", "Justice": "정의",
	"Revenge": "복수", "Betrayal": "배신", "Promise": "약속",
	"Miracle": "기적", "Disaster": "재난", "Chaos": "혼란",

	// 형용사 (감정/상태)
	"Beautiful": "아름다운", "Great": "위대한", "Sad": "슬픈",
	"Happy": "행복한", "Funny": "재미있는", "Scary": "무서운",
	"Dark": "어두운", "Light": "밝은", "Lost": "잃어버린",
	"Found": "찾은", "Last": "마지막", "First": "첫번째",
	"True": "진실된", "False": "거짓된", "Red": "붉은",
	"Blue": "푸른", "Golden": "황금빛", "Silver": "은빛",
	"Silent": "조용한", "Loud": "시끄러운", "Fast": "빠른",
	"Slow": "느린", "Strong": "강한", "Weak": "약한",
	"Young": "젊은", "Old": "늙은",
	"Rich": "부유한", "Poor": "가난한", "Brave": "용감한",
	"Cowardly": "겁쟁이", "Wise": "현명한", "Foolish": "어리석은",
	"Cruel": "잔인한", "Kind": "친절한", "Wild": "야생의",
	"Dangerous": "위험한", "Safe": "안전한",
	"Magic": "마법의", "Ancient": "고대의", "Modern": "현대의",
	"Epic": "서사적인", "Romantic": "낭만적인", "Tragic": "비극적인",
	"Comic": "웃긴", "Fantastic": "환상적인", "Amazing": "놀라운",
	"Incredible": "믿을 수 없는", "Impossible": "불가능한", "Perfect": "완벽한",

	// 동사 (행동)
	"Running": "달리는", "Fighting": "싸우는", "Loving": "사랑하는",
	"Dying": "죽어가는", "Living": "살아있는", "Flying": "날아가는",
	"Falling": "추락하는", "Rising": "떠오르는", "Sleeping": "잠자는",
	"Dreaming": "꿈꾸는", "Thinking": "생각하는", "Knowing": "아는",
	"Seeing": "보는", "Hearing": "듣는", "Speaking": "말하는",
	"Walking": "걷는", "Eating": "먹는", "Drinking": "마시는",
	"Killing": "죽이는", "Saving": "구하는", "Helping": "돕는",
	"Searching": "찾는", "Finding": "발견하는", "Losing": "잃는",
	"Winning": "이기는", "Chasing": "쫓는", "Escaping": "탈출하는",
	"Meeting": "만나는", "Leaving": "떠나는", "Returning": "돌아오는",

	// 장소/자연
	"City": "도시", "Town": "마을", "Village": "시골",
	"Home": "집", "School": "학교", "Office": "사무실",
	"Sea": "바다", "Ocean": "대양", "River": "강",
	"Mountain": "산", "Sky": "하늘", "Star": "별",
	"Sun": "태양", "Moon": "달", "Earth": "지구",
	"Space": "우주", "Forest": "숲", "Desert": "사막",
	"Island": "섬", "Castle": "성", "Prison": "감옥",
	"Hospital": "병원", "Church": "교회", "Garden": "정원",
	"Street": "거리", "Road": "길", "Bridge": "다리",

	// 전치사/접속사/기타
	"About": "에 대한", "With": "와 함께", "Without": "없이",
	"For": "위한", "From": "로부터", "To": "에게",
	"In": "안의", "On": "위의", "At": "에서의",
	"And": "그리고", "But": "그러나", "Or": "또는",
	"If": "만약", "When": "언제", "Where": "어디서",
	"Why": "왜", "How": "어떻게", "Who": "누구",
	"A": "한", "The": "그", "My": "나의",
	"Your": "너의", "Our": "우리의", "Their": "그들의",
	"His": "그의", "Her": "그녀의", "Its": "그것의",
	"This": "이", "That": "저", "Some": "어떤",
	"Many": "많은", "All": "모든", "No": "없는",
}
