// Package auth — single source of truth for password rules.
//
// Rules enforced by ValidatePassword:
//   - length 10..128
//   - not whitespace-only
//   - no leading/trailing whitespace
//   - not in the top-1000 common-passwords blocklist (case-insensitive)
//   - contains at least one letter AND (one digit OR one non-alphanumeric symbol)
//
// MustDifferFrom verifies the new password does not match the current one.
package auth

import (
	"errors"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// MinLength is the shortest password accepted by ValidatePassword.
const MinLength = 10

// MaxLength is the longest password accepted. bcrypt's Go binding
// rejects inputs over 72 bytes with an error rather than the silent
// truncation some older docs describe, so we cap below that hard limit.
// Keeping a 72-byte ceiling also matches the well-known caveat: any
// bytes past 72 are invisible to the hash comparison, so a 200-char
// password and a 73-byte truncation of it would both match the same
// stored hash. Better to reject than to accept a password that the
// server is silently shortening.
const MaxLength = 72

// Score thresholds for the strength gate. Scores below OKThreshold
// are rejected outright on signup/change-password/reset with
// CodePasswordTooWeak. Scores between OK and Strong are accepted but
// the UI shows "ok" instead of "strong". Mirrors the frontend's
// scoring in web/src/lib/password.ts — keep in sync.
const OKScoreThreshold = 40
const StrongScoreThreshold = 70

// PasswordPolicyError codes. Returned as the structured .Err field on
// a kernel-style error so handlers + frontend can switch on them.
const (
	CodePasswordTooShort        = "password_too_short"
	CodePasswordTooLong         = "password_too_long"
	CodePasswordWhitespaceOnly  = "password_whitespace_only"
	CodePasswordLeadingTrailing = "password_has_leading_trailing_space"
	CodePasswordInBlocklist     = "password_in_blocklist"
	CodePasswordNeedsLetter     = "password_needs_letter_and_digit_or_symbol"
	CodePasswordContainsCommon  = "password_contains_common"
	CodePasswordTooWeak         = "password_too_weak"
)

// PasswordPolicyError wraps a violation with a stable error code.
type PasswordPolicyError struct {
	Code string
	Msg  string
}

func (e *PasswordPolicyError) Error() string { return e.Msg }

// IsPasswordPolicyError unwraps to *PasswordPolicyError when present.
func IsPasswordPolicyError(err error) (*PasswordPolicyError, bool) {
	var p *PasswordPolicyError
	if errors.As(err, &p) {
		return p, true
	}
	return nil, false
}

// blocklist is the lowercase, top-1000 common-passwords set from
// SecLists (10k-most-common.txt, lines 1..1000). Lookup is O(1) and
// case-insensitive — we compare against strings.ToLower(plain) and
// against the blocklist keys which are already lowercased.
var blocklist = map[string]struct{}{
	"password":  {},
	"123456":    {},
	"12345678":  {},
	"1234":      {},
	"qwerty":    {},
	"12345":     {},
	"dragon":    {},
	"pussy":     {},
	"baseball":  {},
	"football":  {},
	"letmein":   {},
	"monkey":    {},
	"696969":    {},
	"abc123":    {},
	"mustang":   {},
	"michael":   {},
	"shadow":    {},
	"master":    {},
	"jennifer":  {},
	"111111":    {},
	"2000":      {},
	"jordan":    {},
	"superman":  {},
	"harley":    {},
	"1234567":   {},
	"fuckme":    {},
	"hunter":    {},
	"fuckyou":   {},
	"trustno1":  {},
	"ranger":    {},
	"buster":    {},
	"thomas":    {},
	"tigger":    {},
	"robert":    {},
	"soccer":    {},
	"fuck":      {},
	"batman":    {},
	"test":      {},
	"pass":      {},
	"killer":    {},
	"hockey":    {},
	"george":    {},
	"charlie":   {},
	"andrew":    {},
	"michelle":  {},
	"love":      {},
	"sunshine":  {},
	"jessica":   {},
	"asshole":   {},
	"6969":      {},
	"pepper":    {},
	"daniel":    {},
	"access":    {},
	"123456789": {},
	"654321":    {},
	"joshua":    {},
	"maggie":    {},
	"starwars":  {},
	"silver":    {},
	"william":   {},
	"dallas":    {},
	"yankees":   {},
	"123123":    {},
	"ashley":    {},
	"666666":    {},
	"hello":     {},
	"amanda":    {},
	"orange":    {},
	"biteme":    {},
	"freedom":   {},
	"computer":  {},
	"sexy":      {},
	"thunder":   {},
	"nicole":    {},
	"ginger":    {},
	"heather":   {},
	"hammer":    {},
	"summer":    {},
	"corvette":  {},
	"taylor":    {},
	"fucker":    {},
	"austin":    {},
	"1111":      {},
	"merlin":    {},
	"matthew":   {},
	"121212":    {},
	"golfer":    {},
	"cheese":    {},
	"princess":  {},
	"martin":    {},
	"chelsea":   {},
	"patrick":   {},
	"richard":   {},
	"diamond":   {},
	"yellow":    {},
	"bigdog":    {},
	"secret":    {},
	"asdfgh":    {},
	"sparky":    {},
	"cowboy":    {},
	"camaro":    {},
	"anthony":   {},
	"matrix":    {},
	"falcon":    {},
	"iloveyou":  {},
	"bailey":    {},
	"guitar":    {},
	"jackson":   {},
	"purple":    {},
	"scooter":   {},
	"phoenix":   {},
	"aaaaaa":    {},
	"morgan":    {},
	"tigers":    {},
	"porsche":   {},
	"mickey":    {},
	"maverick":  {},
	"cookie":    {},
	"nascar":    {},
	"peanut":    {},
	"justin":    {},
	"131313":    {},
	"money":     {},
	"horny":     {},
	"samantha":  {},
	"panties":   {},
	"steelers":  {},
	"joseph":    {},
	"snoopy":    {},
	"boomer":    {},
	"whatever":  {},
	"iceman":    {},
	"smokey":    {},
	"gateway":   {},
	"dakota":    {},
	"cowboys":   {},
	"eagles":    {},
	"chicken":   {},
	"dick":      {},
	"black":     {},
	"zxcvbn":    {},
	"please":    {},
	"andrea":    {},
	"ferrari":   {},
	"knight":    {},
	"hardcore":  {},
	"melissa":   {},
	"compaq":    {},
	"coffee":    {},
	"booboo":    {},
	"bitch":     {},
	"johnny":    {},
	"bulldog":   {},
	"xxxxxx":    {},
	"welcome":   {},
	"james":     {},
	"player":    {},
	"ncc1701":   {},
	"wizard":    {},
	"scooby":    {},
	"charles":   {},
	"junior":    {},
	"internet":  {},
	"bigdick":   {},
	"mike":      {},
	"brandy":    {},
	"tennis":    {},
	"blowjob":   {},
	"banana":    {},
	"monster":   {},
	"spider":    {},
	"lakers":    {},
	"miller":    {},
	"rabbit":    {},
	"enter":     {},
	"mercedes":  {},
	"brandon":   {},
	"steven":    {},
	"fender":    {},
	"john":      {},
	"yamaha":    {},
	"diablo":    {},
	"chris":     {},
	"boston":    {},
	"tiger":     {},
	"marine":    {},
	"chicago":   {},
	"rangers":   {},
	"gandalf":   {},
	"winter":    {},
	"bigtits":   {},
	"barney":    {},
	"edward":    {},
	"raiders":   {},
	"porn":      {},
	"badboy":    {},
	"blowme":    {},
	"spanky":    {},
	"bigdaddy":  {},
	"johnson":   {},
	"chester":   {},
	"london":    {},
	"midnight":  {},
	"blue":      {},
	"fishing":   {},
	"000000":    {},
	"hannah":    {},
	"slayer":    {},
	"11111111":  {},
	"rachel":    {},
	"sexsex":    {},
	"redsox":    {},
	"thx1138":   {},
	"asdf":      {},
	"marlboro":  {},
	"panther":   {},
	"zxcvbnm":   {},
	"arsenal":   {},
	"oliver":    {},
	"qazwsx":    {},
	"mother":    {},
	"victoria":  {},
	"7777777":   {},
	"jasper":    {},
	"angel":     {},
	"david":     {},
	"winner":    {},
	"crystal":   {},
	"golden":    {},
	"butthead":  {},
	"viking":    {},
	"jack":      {},
	"iwantu":    {},
	"shannon":   {},
	"murphy":    {},
	"angels":    {},
	"prince":    {},
	"cameron":   {},
	"girls":     {},
	"madison":   {},
	"wilson":    {},
	"carlos":    {},
	"hooters":   {},
	"willie":    {},
	"startrek":  {},
	"captain":   {},
	"maddog":    {},
	"jasmine":   {},
	"butter":    {},
	"booger":    {},
	"angela":    {},
	"golf":      {},
	"lauren":    {},
	"rocket":    {},
	"tiffany":   {},
	"theman":    {},
	"dennis":    {},
	"liverpoo":  {},
	"flower":    {},
	"forever":   {},
	"green":     {},
	"jackie":    {},
	"muffin":    {},
	"turtle":    {},
	"sophie":    {},
	"danielle":  {},
	"redskins":  {},
	"toyota":    {},
	"jason":     {},
	"sierra":    {},
	"winston":   {},
	"debbie":    {},
	"giants":    {},
	"packers":   {},
	"newyork":   {},
	"jeremy":    {},
	"casper":    {},
	"bubba":     {},
	"112233":    {},
	"sandra":    {},
	"lovers":    {},
	"mountain":  {},
	"united":    {},
	"cooper":    {},
	"driver":    {},
	"tucker":    {},
	"helpme":    {},
	"fucking":   {},
	"pookie":    {},
	"lucky":     {},
	"maxwell":   {},
	"8675309":   {},
	"bear":      {},
	"suckit":    {},
	"gators":    {},
	"5150":      {},
	"222222":    {},
	"shithead":  {},
	"fuckoff":   {},
	"jaguar":    {},
	"monica":    {},
	"fred":      {},
	"happy":     {},
	"hotdog":    {},
	"tits":      {},
	"gemini":    {},
	"lover":     {},
	"xxxxxxxx":  {},
	"777777":    {},
	"canada":    {},
	"nathan":    {},
	"victor":    {},
	"florida":   {},
	"88888888":  {},
	"nicholas":  {},
	"rosebud":   {},
	"metallic":  {},
	"doctor":    {},
	"trouble":   {},
	"success":   {},
	"stupid":    {},
	"tomcat":    {},
	"warrior":   {},
	"peaches":   {},
	"apples":    {},
	"fish":      {},
	"qwertyui":  {},
	"magic":     {},
	"buddy":     {},
	"dolphins":  {},
	"rainbow":   {},
	"gunner":    {},
	"987654":    {},
	"freddy":    {},
	"alexis":    {},
	"braves":    {},
	"cock":      {},
	"2112":      {},
	"1212":      {},
	"cocacola":  {},
	"xavier":    {},
	"dolphin":   {},
	"testing":   {},
	"bond007":   {},
	"member":    {},
	"calvin":    {},
	"voodoo":    {},
	"7777":      {},
	"samson":    {},
	"alex":      {},
	"apollo":    {},
	"fire":      {},
	"tester":    {},
	"walter":    {},
	"beavis":    {},
	"voyager":   {},
	"peter":     {},
	"porno":     {},
	"bonnie":    {},
	"rush2112":  {},
	"beer":      {},
	"apple":     {},
	"scorpio":   {},
	"jonathan":  {},
	"skippy":    {},
	"sydney":    {},
	"scott":     {},
	"red123":    {},
	"power":     {},
	"gordon":    {},
	"travis":    {},
	"beaver":    {},
	"star":      {},
	"jackass":   {},
	"flyers":    {},
	"boobs":     {},
	"232323":    {},
	"zzzzzz":    {},
	"steve":     {},
	"rebecca":   {},
	"scorpion":  {},
	"doggie":    {},
	"legend":    {},
	"ou812":     {},
	"yankee":    {},
	"blazer":    {},
	"bill":      {},
	"runner":    {},
	"birdie":    {},
	"bitches":   {},
	"555555":    {},
	"parker":    {},
	"topgun":    {},
	"asdfasdf":  {},
	"heaven":    {},
	"viper":     {},
	"animal":    {},
	"2222":      {},
	"bigboy":    {},
	"4444":      {},
	"arthur":    {},
	"baby":      {},
	"private":   {},
	"godzilla":  {},
	"donald":    {},
	"williams":  {},
	"lifehack":  {},
	"phantom":   {},
	"dave":      {},
	"rock":      {},
	"august":    {},
	"sammy":     {},
	"cool":      {},
	"brian":     {},
	"platinum":  {},
	"jake":      {},
	"bronco":    {},
	"paul":      {},
	"mark":      {},
	"frank":     {},
	"heka6w2":   {},
	"copper":    {},
	"billy":     {},
	"cumshot":   {},
	"garfield":  {},
	"willow":    {},
	"cunt":      {},
	"little":    {},
	"carter":    {},
	"slut":      {},
	"albert":    {},
	"69696969":  {},
	"kitten":    {},
	"super":     {},
	"jordan23":  {},
	"eagle1":    {},
	"shelby":    {},
	"america":   {},
	"11111":     {},
	"jessie":    {},
	"house":     {},
	"free":      {},
	"123321":    {},
	"chevy":     {},
	"bullshit":  {},
	"white":     {},
	"broncos":   {},
	"horney":    {},
	"surfer":    {},
	"nissan":    {},
	"999999":    {},
	"saturn":    {},
	"airborne":  {},
	"elephant":  {},
	"marvin":    {},
	"shit":      {},
	"action":    {},
	"adidas":    {},
	"qwert":     {},
	"kevin":     {},
	"1313":      {},
	"explorer":  {},
	"walker":    {},
	"police":    {},
	"christin":  {},
	"december":  {},
	"benjamin":  {},
	"wolf":      {},
	"sweet":     {},
	"therock":   {},
	"king":      {},
	"online":    {},
	"dickhead":  {},
	"brooklyn":  {},
	"teresa":    {},
	"cricket":   {},
	"sharon":    {},
	"dexter":    {},
	"racing":    {},
	"penis":     {},
	"gregory":   {},
	"0000":      {},
	"teens":     {},
	"redwings":  {},
	"dreams":    {},
	"michigan":  {},
	"hentai":    {},
	"magnum":    {},
	"87654321":  {},
	"nothing":   {},
	"donkey":    {},
	"trinity":   {},
	"digital":   {},
	"333333":    {},
	"stella":    {},
	"cartman":   {},
	"guinness":  {},
	"123abc":    {},
	"speedy":    {},
	"buffalo":   {},
	"kitty":     {},
	"pimpin":    {},
	"eagle":     {},
	"einstein":  {},
	"kelly":     {},
	"nelson":    {},
	"nirvana":   {},
	"vampire":   {},
	"xxxx":      {},
	"playboy":   {},
	"louise":    {},
	"pumpkin":   {},
	"snowball":  {},
	"test123":   {},
	"girl":      {},
	"sucker":    {},
	"mexico":    {},
	"beatles":   {},
	"fantasy":   {},
	"ford":      {},
	"gibson":    {},
	"celtic":    {},
	"marcus":    {},
	"cherry":    {},
	"cassie":    {},
	"888888":    {},
	"natasha":   {},
	"sniper":    {},
	"chance":    {},
	"genesis":   {},
	"hotrod":    {},
	"reddog":    {},
	"alexande":  {},
	"college":   {},
	"jester":    {},
	"passw0rd":  {},
	"bigcock":   {},
	"smith":     {},
	"lasvegas":  {},
	"carmen":    {},
	"slipknot":  {},
	"3333":      {},
	"death":     {},
	"kimberly":  {},
	"1q2w3e":    {},
	"eclipse":   {},
	"1q2w3e4r":  {},
	"stanley":   {},
	"samuel":    {},
	"drummer":   {},
	"homer":     {},
	"montana":   {},
	"music":     {},
	"aaaa":      {},
	"spencer":   {},
	"jimmy":     {},
	"carolina":  {},
	"colorado":  {},
	"creative":  {},
	"hello1":    {},
	"rocky":     {},
	"goober":    {},
	"friday":    {},
	"bollocks":  {},
	"scotty":    {},
	"abcdef":    {},
	"bubbles":   {},
	"hawaii":    {},
	"fluffy":    {},
	"mine":      {},
	"stephen":   {},
	"horses":    {},
	"thumper":   {},
	"5555":      {},
	"pussies":   {},
	"darkness":  {},
	"asdfghjk":  {},
	"pamela":    {},
	"boobies":   {},
	"buddha":    {},
	"vanessa":   {},
	"sandman":   {},
	"naughty":   {},
	"douglas":   {},
	"honda":     {},
	"matt":      {},
	"azerty":    {},
	"6666":      {},
	"shorty":    {},
	"money1":    {},
	"beach":     {},
	"loveme":    {},
	"4321":      {},
	"simple":    {},
	"poohbear":  {},
	"444444":    {},
	"badass":    {},
	"destiny":   {},
	"sarah":     {},
	"denise":    {},
	"vikings":   {},
	"lizard":    {},
	"melanie":   {},
	"assman":    {},
	"sabrina":   {},
	"nintendo":  {},
	"water":     {},
	"good":      {},
	"howard":    {},
	"time":      {},
	"123qwe":    {},
	"november":  {},
	"xxxxx":     {},
	"october":   {},
	"leather":   {},
	"bastard":   {},
	"young":     {},
	"101010":    {},
	"extreme":   {},
	"hard":      {},
	"password1": {},
	"vincent":   {},
	"pussy1":    {},
	"lacrosse":  {},
	"hotmail":   {},
	"spooky":    {},
	"amateur":   {},
	"alaska":    {},
	"badger":    {},
	"paradise":  {},
	"maryjane":  {},
	"poop":      {},
	"crazy":     {},
	"mozart":    {},
	"video":     {},
	"russell":   {},
	"vagina":    {},
	"spitfire":  {},
	"anderson":  {},
	"norman":    {},
	"eric":      {},
	"cherokee":  {},
	"cougar":    {},
	"barbara":   {},
	"long":      {},
	"420420":    {},
	"family":    {},
	"horse":     {},
	"enigma":    {},
	"allison":   {},
	"raider":    {},
	"brazil":    {},
	"blonde":    {},
	"jones":     {},
	"55555":     {},
	"dude":      {},
	"drowssap":  {},
	"jeff":      {},
	"school":    {},
	"marshall":  {},
	"lovely":    {},
	"1qaz2wsx":  {},
	"jeffrey":   {},
	"caroline":  {},
	"franklin":  {},
	"booty":     {},
	"molly":     {},
	"snickers":  {},
	"leslie":    {},
	"nipples":   {},
	"courtney":  {},
	"diesel":    {},
	"rocks":     {},
	"eminem":    {},
	"westside":  {},
	"suzuki":    {},
	"daddy":     {},
	"passion":   {},
	"hummer":    {},
	"ladies":    {},
	"zachary":   {},
	"frankie":   {},
	"elvis":     {},
	"reggie":    {},
	"alpha":     {},
	"suckme":    {},
	"simpson":   {},
	"patricia":  {},
	"147147":    {},
	"pirate":    {},
	"tommy":     {},
	"semperfi":  {},
	"jupiter":   {},
	"redrum":    {},
	"freeuser":  {},
	"wanker":    {},
	"stinky":    {},
	"ducati":    {},
	"paris":     {},
	"natalie":   {},
	"babygirl":  {},
	"bishop":    {},
	"windows":   {},
	"spirit":    {},
	"pantera":   {},
	"monday":    {},
	"patches":   {},
	"brutus":    {},
	"houston":   {},
	"smooth":    {},
	"penguin":   {},
	"marley":    {},
	"forest":    {},
	"cream":     {},
	"212121":    {},
	"flash":     {},
	"maximus":   {},
	"nipple":    {},
	"bobby":     {},
	"bradley":   {},
	"vision":    {},
	"pokemon":   {},
	"champion":  {},
	"fireman":   {},
	"indian":    {},
	"softball":  {},
	"picard":    {},
	"system":    {},
	"clinton":   {},
	"cobra":     {},
	"enjoy":     {},
	"lucky1":    {},
	"claire":    {},
	"claudia":   {},
	"boogie":    {},
	"timothy":   {},
	"marines":   {},
	"security":  {},
	"dirty":     {},
	"admin":     {},
	"wildcats":  {},
	"pimp":      {},
	"dancer":    {},
	"hardon":    {},
	"veronica":  {},
	"fucked":    {},
	"abcd1234":  {},
	"abcdefg":   {},
	"ironman":   {},
	"wolverin":  {},
	"remember":  {},
	"great":     {},
	"freepass":  {},
	"bigred":    {},
	"squirt":    {},
	"justice":   {},
	"francis":   {},
	"hobbes":    {},
	"kermit":    {},
	"pearljam":  {},
	"mercury":   {},
	"domino":    {},
	"9999":      {},
	"denver":    {},
	"brooke":    {},
	"rascal":    {},
	"hitman":    {},
	"mistress":  {},
	"simon":     {},
	"tony":      {},
	"bbbbbb":    {},
	"friend":    {},
	"peekaboo":  {},
	"naked":     {},
	"budlight":  {},
	"electric":  {},
	"sluts":     {},
	"stargate":  {},
	"saints":    {},
	"bondage":   {},
	"brittany":  {},
	"bigman":    {},
	"zombie":    {},
	"swimming":  {},
	"duke":      {},
	"qwerty1":   {},
	"babes":     {},
	"scotland":  {},
	"disney":    {},
	"rooster":   {},
	"brenda":    {},
	"mookie":    {},
	"swordfis":  {},
	"candy":     {},
	"duncan":    {},
	"olivia":    {},
	"hunting":   {},
	"blink182":  {},
	"alicia":    {},
	"8888":      {},
	"samsung":   {},
	"bubba1":    {},
	"whore":     {},
	"virginia":  {},
	"general":   {},
	"passport":  {},
	"aaaaaaaa":  {},
	"erotic":    {},
	"liberty":   {},
	"arizona":   {},
	"jesus":     {},
	"abcd":      {},
	"newport":   {},
	"skipper":   {},
	"rolltide":  {},
	"balls":     {},
	"happy1":    {},
	"galore":    {},
	"christ":    {},
	"weasel":    {},
	"242424":    {},
	"wombat":    {},
	"digger":    {},
	"classic":   {},
	"bulldogs":  {},
	"poopoo":    {},
	"accord":    {},
	"popcorn":   {},
	"turkey":    {},
	"jenny":     {},
	"amber":     {},
	"bunny":     {},
	"mouse":     {},
	"007007":    {},
	"titanic":   {},
	"liverpool": {},
	"dreamer":   {},
	"everton":   {},
	"friends":   {},
	"chevelle":  {},
	"carrie":    {},
	"gabriel":   {},
	"psycho":    {},
	"nemesis":   {},
	"burton":    {},
	"pontiac":   {},
	"connor":    {},
	"eatme":     {},
	"lickme":    {},
	"roland":    {},
	"cumming":   {},
	"mitchell":  {},
	"ireland":   {},
	"lincoln":   {},
	"arnold":    {},
	"spiderma":  {},
	"patriots":  {},
	"goblue":    {},
	"devils":    {},
	"eugene":    {},
	"empire":    {},
	"asdfg":     {},
	"cardinal":  {},
	"brown":     {},
	"shaggy":    {},
	"froggy":    {},
	"qwer":      {},
	"kawasaki":  {},
	"kodiak":    {},
	"people":    {},
	"phpbb":     {},
	"light":     {},
	"54321":     {},
	"kramer":    {},
	"chopper":   {},
	"hooker":    {},
	"honey":     {},
	"whynot":    {},
	"lesbian":   {},
	"lisa":      {},
	"baxter":    {},
	"adam":      {},
	"snake":     {},
	"teen":      {},
	"ncc1701d":  {},
	"qqqqqq":    {},
	"airplane":  {},
	"britney":   {},
	"avalon":    {},
	"sandy":     {},
	"sugar":     {},
	"sublime":   {},
	"stewart":   {},
	"wildcat":   {},
	"raven":     {},
	"scarface":  {},
	"elizabet":  {},
	"123654":    {},
	"trucks":    {},
	"wolfpack":  {},
	"pervert":   {},
	"lawrence":  {},
	"raymond":   {},
	"redhead":   {},
	"american":  {},
	"alyssa":    {},
	"bambam":    {},
	"movie":     {},
	"woody":     {},
	"shaved":    {},
	"snowman":   {},
	"tiger1":    {},
	"chicks":    {},
	"raptor":    {},
	"1969":      {},
	"stingray":  {},
	"shooter":   {},
	"france":    {},
	"stars":     {},
	"madmax":    {},
	"kristen":   {},
	"sports":    {},
	"jerry":     {},
	"789456":    {},
	"garcia":    {},
	"simpsons":  {},
	"lights":    {},
	"ryan":      {},
	"looking":   {},
	"chronic":   {},
	"alison":    {},
	"hahaha":    {},
	"packard":   {},
	"hendrix":   {},
	"perfect":   {},
	"service":   {},
	"spring":    {},
	"srinivas":  {},
	"spike":     {},
	"katie":     {},
	"252525":    {},
	"oscar":     {},
	"brother":   {},
	"bigmac":    {},
	"suck":      {},
	"single":    {},
	"cannon":    {},
	"georgia":   {},
	"popeye":    {},
	"tattoo":    {},
	"texas":     {},
	"party":     {},
	"bullet":    {},
	"taurus":    {},
	"sailor":    {},
	"wolves":    {},
	"panthers":  {},
	"japan":     {},
	"strike":    {},
	"flowers":   {},
	"pussycat":  {},
	"chris1":    {},
	"loverboy":  {},
	"berlin":    {},
	"sticky":    {},
	"marina":    {},
	"tarheels":  {},
	"fisher":    {},
	"russia":    {},
	"connie":    {},
	"wolfgang":  {},
	"testtest":  {},
	"mature":    {},
	"bass":      {},
	"catch22":   {},
	"juice":     {},
	"michael1":  {},
	"nigger":    {},
	"159753":    {},
	"women":     {},
	"alpha1":    {},
	"trooper":   {},
	"hawkeye":   {},
	"head":      {},
	"freaky":    {},
	"dodgers":   {},
	"pakistan":  {},
	"machine":   {},
	"pyramid":   {},
	"vegeta":    {},
	"katana":    {},
	"moose":     {},
	"tinker":    {},
	"coyote":    {},
	"infinity":  {},
	"inside":    {},
	"pepsi":     {},
	"letmein1":  {},
	"bang":      {},
	"control":   {},
}

func inBlocklist(plain string) bool {
	_, ok := blocklist[strings.ToLower(plain)]
	return ok
}

// containsCommonSubstring reports whether plain contains any entry
// from the top-1000 blocklist as a case-insensitive substring of
// length >= 4. Catches "passwordpassword" (contains "password"),
// "qwertyqwerty", "adminadmin123" etc. that the exact-match
// blocklist would miss.
//
// 1000 entries * average word length ~7 chars = ~7000 char
// comparisons worst case. Fast enough to run on every signup /
// change-password / reset-password request.
func containsCommonSubstring(plain string) bool {
	lower := strings.ToLower(plain)
	if len(lower) == 0 {
		return false
	}
	for word := range blocklist {
		if len(word) >= 4 && strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

// ScorePassword returns a 0-100 score that approximates the actual
// entropy / guessability of plain. Same algorithm as the frontend
// (web/src/lib/password.ts scorePassword) — keep in sync.
//
// Components:
//   - Length tier (0-40 pts): longer is better, with diminishing returns.
//   - Character diversity (0-25 pts): lowercase/uppercase/digit/symbol.
//   - Uniqueness (0-25 pts): uniqueChars / length, scaled.
//   - No obvious patterns (0-10 pts): no 4+ char runs, not in common-passwords.
func ScorePassword(plain string) int {
	pwd := plain
	if len(pwd) == 0 {
		return 0
	}

	// 1. Length tier (0-40 pts).
	lengthScore := 0
	switch {
	case len(pwd) >= 10 && len(pwd) <= 11:
		lengthScore = 15
	case len(pwd) <= 13:
		lengthScore = 22
	case len(pwd) <= 15:
		lengthScore = 28
	case len(pwd) <= 19:
		lengthScore = 33
	case len(pwd) <= 31:
		lengthScore = 38
	default:
		lengthScore = 40
	}

	// 2. Character diversity (0-25 pts).
	diversityScore := 0
	hasLower, hasUpper, hasDigit, hasSymbol := false, false, false, false
	for _, r := range pwd {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if hasLower {
		diversityScore += 6
	}
	if hasUpper {
		diversityScore += 6
	}
	if hasDigit {
		diversityScore += 6
	}
	if hasSymbol {
		diversityScore += 6
	}
	if hasLower && hasUpper && hasDigit && hasSymbol {
		diversityScore++
	}

	// 3. Uniqueness (0-25 pts). Penalize runs and low diversity
	// heavily. "hhhhhhhhhhhhhh1" has 2 unique chars out of 15
	// (ratio 0.13) → ~3 pts. We use a quadratic curve so that very
	// low ratios (< 0.3) get crushed: uniqueRatio^1.5 * 25.
	seen := make(map[rune]struct{}, len(pwd))
	for _, r := range pwd {
		seen[r] = struct{}{}
	}
	uniqueRatio := float64(len(seen)) / float64(len(pwd))
	uniquenessScore := int(math.Pow(uniqueRatio, 1.5) * 25)

	// 4. No obvious patterns (0-10 pts).
	patternScore := 0
	if !hasLongRun(pwd) {
		patternScore += 5
	}
	if !inBlocklist(pwd) {
		patternScore += 5
	}

	return lengthScore + diversityScore + uniquenessScore + patternScore
}

// hasLongRun reports whether plain contains a run of 4+ identical
// characters in a row (e.g. "hhhh", "1111", "aaaa"). Simple regex
// because that's all we need.
func hasLongRun(plain string) bool {
	for i := 0; i+3 < len(plain); i++ {
		if plain[i] == plain[i+1] && plain[i+1] == plain[i+2] && plain[i+2] == plain[i+3] {
			return true
		}
	}
	return false
}

// ValidatePassword applies the full policy and returns the first
// violation found. nil = password is acceptable.
func ValidatePassword(plain string) error {
	if utf8.RuneCountInString(plain) < MinLength {
		return &PasswordPolicyError{
			Code: CodePasswordTooShort,
			Msg:  "password must be at least 10 characters",
		}
	}
	if utf8.RuneCountInString(plain) > MaxLength {
		return &PasswordPolicyError{
			Code: CodePasswordTooLong,
			Msg:  "password must be 128 characters or fewer",
		}
	}
	if strings.TrimSpace(plain) == "" {
		return &PasswordPolicyError{
			Code: CodePasswordWhitespaceOnly,
			Msg:  "password cannot be only spaces",
		}
	}
	if plain != strings.TrimSpace(plain) {
		return &PasswordPolicyError{
			Code: CodePasswordLeadingTrailing,
			Msg:  "remove the spaces at the start and end of the password",
		}
	}
	if inBlocklist(plain) {
		return &PasswordPolicyError{
			Code: CodePasswordInBlocklist,
			Msg:  "that password is too common; pick something less guessable",
		}
	}
	if containsCommonSubstring(plain) {
		return &PasswordPolicyError{
			Code: CodePasswordContainsCommon,
			Msg:  "that password contains a commonly-used word; try mixing it up",
		}
	}
	if !hasLetterAndDigitOrSymbol(plain) {
		return &PasswordPolicyError{
			Code: CodePasswordNeedsLetter,
			Msg:  "add at least one letter and either a digit or a symbol",
		}
	}
	if ScorePassword(plain) < OKScoreThreshold {
		return &PasswordPolicyError{
			Code: CodePasswordTooWeak,
			Msg:  "that password is too simple — try mixing in some variety (different characters, no repeats, no obvious patterns)",
		}
	}
	return nil
}

// MustDifferFrom verifies that plain is not the same password as the
// one currently stored. Pass the existing bcrypt hash.
func MustDifferFrom(existingHash, plain string) error {
	if bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(plain)) == nil {
		return &PasswordPolicyError{
			Code: "password_must_differ",
			Msg:  "new password must be different from your current password",
		}
	}
	return nil
}

func hasLetterAndDigitOrSymbol(s string) bool {
	hasLetter := false
	hasDigitOrSymbol := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r), unicode.IsPunct(r), unicode.IsSymbol(r):
			hasDigitOrSymbol = true
		}
		if hasLetter && hasDigitOrSymbol {
			return true
		}
	}
	return false
}
