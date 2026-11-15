# Implementation plan - Key-Value engine (NASP 2026/27) - cilj: ocena 10

> **Ovaj fajl je zajednička referenca za ceo tim i za agente koje svako od nas koristi.**
> Izmene ovog fajla idu isključivo kroz Pull Request koji odobre bar dva člana tima.

**Tim:** Andrej, Aleksa, Đole, Relja
**Repo:** https://github.com/cupinaa/napredni-algoritmi-projekat
**Jezik:** Go (1.22+)
**Specifikacija:** `Specifikacija.pdf` / `Specifikacija.md` (projektni zadatak 2026/2027) + materijali sa vežbi 1-12
**Cilj:** ocena **10** - svi zahtevi iz specifikacije, uključujući sve dodatne zahteve (DZ).

**Kako se penjemo:** svaki član ima svoj spisak koraka poređan od lakšeg ka težem (sekcija 5). Korak je označen ocenom kojoj pripada. Ceo tim zajedno završava sve korake za ocenu 6, pa za 7, i tako redom do 10 - dodatni zadaci dolaze na kraju svakog spiska. Posle svake ocene projekat radi i može da se brani (tag na `main`).

---

## 0. Način rada

**Kod pišemo sami.** Agent je mentor i sagovornik, ne autor:

- Mi donosimo ideju i pristup, agent daje **kritičko mišljenje** (šta ne valja, šta je bolje, šta specifikacija traži), i zajedno dolazimo do rešenja.
- Agent nam daje **blokove koda** (jer još ne znamo Go sintaksu dovoljno dobro) i objašnjava ih. Mi ih **ručno prekucavamo** u fajlove i foldere koje smo **sami napravili**. Agent ne pravi i ne menja fajlove u repou.
- Svrha: da svako zna strukturu projekta i svaku liniju u svojim fajlovima - na odbrani dobijamo eliminacioni zadatak koji menja kod uživo.

**Pravila projekta (iz specifikacije - važe za sav kod):**

1. **Zabranjene biblioteke za serijalizaciju.** Dozvoljeno: `encoding/binary`, `hash/crc32`, `math`, `bytes` i slično što pretvara brojeve u bajtove. Zabranjeno: `encoding/gob`, `encoding/json` (osim za konfiguracioni fajl), protobuf, msgpack i sve slično. Sve binarne formate pišemo ručno.
2. **Sadržaj fajlova se čita i piše ISKLJUČIVO preko Block managera** (`internal/blokmenadzer`). Izuzeci: učitavanje konfiguracionog fajla i operacije nad fajl-sistemom koje ne čitaju sadržaj (pravljenje direktorijuma, listanje, preimenovanje, brisanje fajla, hard link).
3. **Ništa se ne učitava ceo u memoriju** gde specifikacija kaže "blok po blok" (WAL, Data, Index, Summary).

**Konvencije koda (dogovor tima):**

- Opisna imena na srpskom (latinica, bez dijakritika): `kljuc`, `vrednost`, `brojac`, `generacija`, `NadjiZapis`. Bez skraćenica tipa `cnt`, `idx`, `tmp`. Domenski pojmovi ostaju kako jesu (`WAL`, `SSTable`, `Memtable`, `Bloom`, `CRC`, `LSM`).
- Svaka javna funkcija ima komentar: šta radi, šta vraća, kada vraća grešku. Za algoritme koje specifikacija traži da se obrazlože (scan, iteratori, izbor kompresije) komentar objašnjava **zašto** je izabran taj pristup.
- Svaki paket ima testove (`_test.go`). Pre commit-a: `go vet ./...` i `go test ./...`.

---

## 1. Kratak opis sistema

Key-value engine kao konzolna aplikacija. Ključ je `string`, vrednost je `[]byte`.

**Write path (PUT / DELETE):**
WAL (zapis na disk) → Memtable (memorija) → kad se Memtable napuni: sortiraj i napravi SSTable → proveri uslov za kompakciju (od ocene 9, može lančano da pokrene kompakcije na narednim nivoima).

**Read path (GET):**
Memtable → Cache (od ocene 7) → SSTable-ovi od najnovijeg ka najstarijem (od ocene 9: nivo po nivo, redosledom koji određuje algoritam kompakcije): Bloom filter → Summary (opseg + pozicija u Indexu) → Index (pozicija u Data) → Data.

**Brisanje** je logičko: upisuje se zapis sa `Tombstone = true`. Na read path-u **prvi pronađeni zapis za ključ je merodavan** - ako je tombstone (ili mu je istekao TTL), ključ ne postoji i pretraga se prekida.

---

## 2. Git workflow (GitFlow, kako je rađeno na vežbama 2)

### 2.1 Grane

| Grana | Namena | Ko piše u nju |
|---|---|---|
| `main` | Samo stabilne verzije, po jedna za svaku završenu ocenu (tag `ocena-6` … `ocena-10`) | Samo merge iz `develop` preko PR-a |
| `develop` | Integraciona grana - ovde se spaja sav završen rad | Samo merge iz `feature/*` i `fix/*` preko PR-a |
| `feature/<naziv>` | Jedan korak iz sekcije 5 | Vlasnik koraka |
| `fix/<naziv>` | Ispravka greške | Bilo ko |

Direktan push na `main` i `develop` je zabranjen (izuzetak: prvi commit sa ovim planom).

### 2.2 Tok rada za jedan korak

```bash
git checkout develop
git pull origin develop
git checkout -b feature/blok-menadzer      # naziv grane iz sekcije 5
# ... rad, mali commit-ovi ...
git push -u origin feature/blok-menadzer
# Pre otvaranja PR-a, pokupi najnovije iz develop-a:
git fetch origin
git merge origin/develop                   # reši konflikte lokalno, pa pokreni testove
git push
# Otvori PR: feature/blok-menadzer → develop
```

**Pravila za PR:**
- Bar **jedan reviewer koji nije autor** (rotacija u sekciji 5.6). Reviewer zaista čita kod - to je i priprema za odbranu.
- CI (GitHub Actions: `go vet` + `go test`) mora da prođe.
- Merge se radi kao **običan merge commit, NE squash** - asistent mora da vidi kontinualan rad svakog člana kroz istoriju commit-ova.
- Posle merge-a grana se briše.

### 2.3 Commit poruke

Format: `<prefiks>: <kratak opis na srpskom>`

Prefiksi: `add` (nova funkcionalnost), `fix` (ispravka), `refactor`, `test`, `docs`, `chore` (konfiguracija, CI).

Primeri: `add: citanje bloka kroz block cache`, `fix: CRC provera preskakala poslednji zapis`, `test: oporavak Memtable-a iz WAL-a`.

Commit-uj i push-uj **bar nekoliko puta nedeljno** - repo mora da pokazuje kontinualan rad.

### 2.4 Zaštita grana (radi vlasnik repoa, jednom)

GitHub → Settings → Branches: za `main` i `develop` uključiti "Require a pull request before merging" (1 approval) i "Require status checks to pass".

---

## 3. Arhitektura i zajednički ugovori

> Sve u ovoj sekciji dogovaramo u **Fazi 0**, zajedno, pre nego što iko krene na svoj korak. Posle toga je **zamrznuto**: menja se samo dogovorom celog tima i PR-om na ovaj fajl.

Dve odluke donosimo odmah iako trebaju tek za ocenu 10, jer bi ih kasnije bilo skupo uvoditi:
- **Format zapisa odmah ima polja `Tip` i `Istice`** (za probabilističke tipove, BLOB i TTL) - inače bi se format WAL-a i SSTable-a menjao usred projekta.
- **Engine od početka podržava keyspace-ove** (`map[string]*Prostor`), sa jednim prostorom `"default"` - inače bi 2.9 bio refaktor celog engine-a.

### 3.1 Struktura repozitorijuma

```
napredni-algoritmi-projekat/
├── cmd/kvengine/main.go          # konzolna aplikacija (CLI petlja, parsiranje komandi)
├── internal/
│   ├── konfiguracija/            # učitavanje config.json + default vrednosti + validacija
│   ├── zapis/                    # struktura Zapis + binarno kodiranje/dekodiranje (zajedničko!)
│   ├── blokmenadzer/             # Block manager, block cache, tok bajtova preko blokova
│   ├── lru/                      # generički LRU (koriste ga block cache i cache zapisa)
│   ├── wal/                      # segmentirani Write-Ahead Log
│   ├── memtable/                 # interfejs + hashmapa, skiplista, bstablo + menadžer N instanci
│   ├── bloomfilter/              # koristi ga SSTable Filter i tip BloomFilter (2.3)
│   ├── merkle/
│   ├── kodiranje/                # kompresovani format data bloka (ocena 9)
│   ├── sstable/                  # pisanje/čitanje SSTable-a + iterator
│   ├── kes/                      # cache zapisa na read path-u (1.5)
│   ├── lsm/                      # nivoi, kompakcije, MANIFEST
│   ├── iteratori/                # merge iterator, PREFIX/RANGE scan i iterate
│   ├── tokenbucket/
│   ├── probabilisticki/          # CountMinSketch, HyperLogLog, SimHash (strukture + komande)
│   ├── mergeoperatori/           # merge operatori za BF/CMS/HLL (2.3)
│   ├── blob/                     # BLOB fajlovi (2.8)
│   ├── kopije/                   # snapshot, backup, inkrementalni backup, restore (Relja); checkpoint.go (Đole)
│   └── engine/                   # spaja sve: read/write path, keyspace-ovi, sistemski ključevi, TTL
├── config.json                   # primer konfiguracije sa default vrednostima
├── .github/workflows/ci.yml
├── README.md                     # komande + obrazloženja algoritama (scan, iteratori, kompresija, kompakcije)
└── <ovaj plan>
```

Go modul: `github.com/cupinaa/napredni-algoritmi-projekat`

### 3.2 Format zapisa

**Logički zapis (`internal/zapis`):**

```go
type TipZapisa uint8

const (
    TipObican         TipZapisa = 0 // obična vrednost
    TipMergeOperacija TipZapisa = 1 // operacija nad BloomFilter/CMS/HLL instancom (2.3, ocena 10)
    TipBlobReferenca  TipZapisa = 2 // vrednost je u BLOB fajlu (2.8, ocena 10)
)

type Zapis struct {
    Kljuc     string
    Vrednost  []byte
    Timestamp int64     // time.Now().UnixNano() - nanosekunde, da dva PUT-a u istoj sekundi ne budu izjednačena
    Tombstone bool
    Tip       TipZapisa // do ocene 10 uvek TipObican
    Istice    int64     // UnixNano kada zapis ističe; 0 = nema TTL (2.7, ocena 10)
}
```

**WAL zapis - osnovni (ocena 6), format sa vežbi 7 proširen sa `Tip` i `Istice`:**

```
+---------+---------------+---------------+---------+-------------+---------------+-----------------+-----+-------+
| CRC (4B)| Timestamp (8B)| Tombstone (1B)| Tip (1B)| Istice (8B) | Key Size (8B) | Value Size (8B) | Key | Value |
+---------+---------------+---------------+---------+-------------+---------------+-----------------+-----+-------+
CRC = crc32.ChecksumIEEE nad svim bajtovima posle CRC polja. Brojevi little-endian (encoding/binary).
```

**WAL zapis - blokovski (ocena 8, 1.1[DZ1]):** dodaju se polje **Type** (FULL / FIRST / MIDDLE / LAST) za fragmentaciju korisničkog zapisa preko granica blokova, **padding** do kraja bloka i **trailing canary** polje na kraju svakog zapisa. Tačan raspored Aleksa upisuje ovde kad dođe do tog koraka.

**SSTable Data zapis:** do ocene 9 isti raspored polja kao osnovni WAL zapis. Od ocene 9 (1.3[DZ3]) prelazi na kompresovani format iz paketa `kodiranje` (varint, delta ključeva, dictionary/bit packing) - Aleksa upisuje raspored ovde. Da bi ta zamena bila mala, **SSTable kodira i dekodira zapise na jednom mestu** (jedna funkcija za upis, jedna za čitanje) - u ocenu 9 se menja samo to mesto.

### 3.3 Interfejsi između komponenti

Potpisi koje svi poštuju. Metode označene sa (ocena N) dodaju se kad se dođe do tog koraka.

```go
// internal/blokmenadzer
type BlokMenadzer interface {
    CitajBlok(putanja string, redniBroj uint64) ([]byte, error)          // prvo block cache, pa disk
    PisiBlok(putanja string, redniBroj uint64, sadrzaj []byte) error     // disk + ažurira cache ako je blok tu
    BrojBlokova(putanja string) (uint64, error)
    Sinhronizuj(putanja string) error                                    // fdatasync / File.Sync (ocena 8)
    VelicinaBloka() int
}

// Pomoćni sloj (isti paket): niz bajtova proizvoljne dužine preko blokova.
// SSTable i WAL ga koriste da ne bi svako sam računao granice blokova.
type PisacToka interface {
    Upisi(podaci []byte) (pocetniOffset uint64, err error)
    Zavrsi() error // dopuni poslednji blok i upiše ga
}
type CitacToka interface {
    ProcitajNa(offset uint64, duzina uint64) ([]byte, error) // čita samo blokove koji pokrivaju traženi opseg
}

// internal/memtable
type Memtable interface {
    Dodaj(z zapis.Zapis)                       // PUT i DELETE (tombstone); isti ključ = prepisivanje
    Nadji(kljuc string) (zapis.Zapis, bool)   // vraća i tombstone zapise - engine odlučuje šta znače
    BrojElemenata() int
    ZauzeceBajtova() int
    SortiraniZapisi() []zapis.Zapis           // za flush, rastuće po ključu
    Iterator(odKljuca string) Iterator         // (ocena 9) za scan i iteratore
}

// internal/wal
type WAL interface {
    Dodaj(zapisi ...zapis.Zapis) (redniBrojSegmenta uint64, err error) // vraća tek kad je na disku; više zapisa = atomičan batch (ocena 10)
    Procitaj(posetilac func(z zapis.Zapis) error) error                // blok po blok, proverava CRC
    ObrisiSegmenteDo(niskaVoda uint64) error                           // briše segmente čiji su zapisi već u SSTable-u
}

// internal/sstable
func Napravi(bm blokmenadzer.BlokMenadzer, direktorijum string, generacija uint64,
    zapisi []zapis.Zapis, opcije Opcije) (*Tabela, error)
func Otvori(bm blokmenadzer.BlokMenadzer, direktorijum string, generacija uint64) (*Tabela, error)
func (t *Tabela) Nadji(kljuc string) (zapis.Zapis, bool, error)   // Bloom → Summary → Index → Data
func (t *Tabela) Validiraj() (izmenjeniZapisi []int, err error)    // (ocena 7) Merkle provera
func (t *Tabela) Iterator(odKljuca string) (Iterator, error)      // (ocena 8) blok po blok

// Zajednički iterator (memtable, sstable, merge iterator)
type Iterator interface {
    Sledeci() (zapis.Zapis, bool, error)
    Zatvori() error
}

// internal/lru (generički)
func Novi[K comparable, V any](kapacitet int) *LRU[K, V]
func (l *LRU[K, V]) Uzmi(kljuc K) (V, bool)
func (l *LRU[K, V]) Stavi(kljuc K, vrednost V)
func (l *LRU[K, V]) Izbaci(kljuc K)

// internal/engine
type Engine struct { /* map[string]*Prostor - jedan Prostor po keyspace-u */ }
func (e *Engine) Put(prostor, kljuc string, vrednost []byte) error
func (e *Engine) Get(prostor, kljuc string) ([]byte, bool, error)
func (e *Engine) Delete(prostor, kljuc string) error
```

**Sistemski ključevi** (token bucket 2.2, probabilistički tipovi 2.3): čuvaju se kao obični zapisi sa prefiksom `"\x00sys/"`. Engine odbija svaku korisničku PUT/GET/DELETE operaciju nad ključem koji počinje sa `"\x00"`, a scan/iterate ih preskaču.

### 3.4 Fajlovi na disku

```
podaci/
└── <keyspace>/                   # "default" dok se ne uvedu keyspace-ovi
    ├── wal/wal_0001.log, wal_0002.log, ...
    ├── sstable/usertable-<GEN>-<ELEMENT>.db     # ELEMENT: Data, Filter, Index, Summary, Metadata
    │          usertable-<GEN>-TOC.txt           # spisak elemenata tabele (vežbe 9)
    ├── blob/                     # ocena 10
    └── MANIFEST-<n>              # ocena 10
checkpoint/<ime>/   backup/<ime>/
```

`GEN` je redni broj koji raste sa svakom novom tabelom (nikad se ne ponavlja). Nivo LSM-a se do MANIFEST-a piše u imenu (`usertable-<GEN>-L<nivo>-Data.db`), a od MANIFEST-a se vodi u njemu.

### 3.5 Konfiguracija (`config.json`)

Format je **JSON** (`encoding/json` je dozvoljen samo ovde). Ako fajl ne postoji ili neko polje fali, koristi se default iz koda. Nevalidne vrednosti (npr. blok od 5 KB) prijavljuju grešku pri pokretanju. Svaki korak koji uvodi novo podešavanje dodaje ga ovde i u default vrednosti.

```json
{
  "blok":         { "velicina_kb": 4, "kes_blokova": 256 },
  "wal":          { "zapisa_po_segmentu": 100, "blokova_po_segmentu": 16 },
  "memtable":     { "struktura": "hashmapa", "max_elemenata": 1000, "max_kb": 0, "broj_instanci": 1,
                    "skiplista_max_visina": 16, "bstablo_red": 4 },
  "sstable":      { "proredjenost_summary": 5, "bloom_verovatnoca_greske": 0.01 },
  "kes":          { "max_zapisa": 100 },
  "lsm":          { "max_nivoa": 4, "algoritam": "size_tiered",
                    "size_tiered_tabela_za_spajanje": 4,
                    "leveled_l0_max_tabela": 4, "leveled_l1_max_kb": 1024, "leveled_faktor_rasta": 10,
                    "hibridni_size_tiered_nivoa": 1 },
  "token_bucket": { "broj_tokena": 10, "interval_sekundi": 60 },
  "blob":         { "prag_kb": 64 },
  "putanje":      { "podaci": "podaci", "checkpoint": "checkpoint", "backup": "backup" }
}
```

Pravila: `max_kb = 0` znači da se Memtable ograničava brojem elemenata, inače zauzećem memorije. `blok.velicina_kb` ∈ {4, 8, 16}. `lsm.algoritam` ∈ {`size_tiered`, `leveled`, `hibridni`}. `memtable.struktura` ∈ {`hashmapa`, `skiplista`, `bstablo`}. `lsm.max_nivoa` ≥ 4 (vežbe 11).

### 3.6 Komande u konzoli

```
PUT <kljuc> <vrednost>      GET <kljuc>      DELETE <kljuc>      EXIT                     (ocena 6)
VALIDATE <generacija>                                                                    (ocena 7)
PREFIX_SCAN <prefiks> <strana> <velicina>    RANGE_SCAN <od> <do> <strana> <velicina>
PREFIX_ITERATE <prefiks> / RANGE_ITERATE <od> <do>  → zatim NEXT / STOP
SNAPSHOT ...   CHECKPOINT <ime>                                                          (ocena 9)
USE <keyspace>   PUT <kljuc> <vrednost> [ttl_sekundi]   BATCH_WRITE ...   DELETE_RANGE <od> <do>
BACKUP <ime>   BACKUP_INC <ime>   RESTORE <ime>
BF_* / CMS_* / HLL_* / SIMHASH_*                                                         (ocena 10)
```

Tačnu sintaksu svake komande definiše vlasnik koraka i upisuje je u `README.md`. Svaki vlasnik sam dodaje svoje komande u CLI (u dogovoru sa Reljom, vlasnikom CLI-ja).

---

## 4. Faze i rokovi

Faza = ocena. Faza je gotova kad svi članovi završe svoje korake označene tom ocenom.

> **Datum odbrane u februarskom roku još nije poznat - kad se sazna, upisati ga ovde i pomeriti rokove.** Do tada su datumi okvirni.

| Faza | Ocena | Specifikacija | Okvirni rok |
|---|---|---|---|
| 0 | - | Skelet, zamrznuti ugovori iz sekcije 3, CI, svi umeju da prave grane i PR-ove | 22.11.2026. |
| 1 | 6 | 1.1, 1.2 (hash mapa), 1.6, 1.3 bez 1.3.5, 2.1, PUT/GET/DELETE | 13.12.2026. |
| 2 | 7 | 1.3.5, 1.5, 1.2[DZ1], 1.3[DZ1] | 27.12.2026. |
| 3 | 8 | 2.2, 1.1[DZ1], 1.2[DZ2] | 10.01.2027. |
| 4 | 9 | 1.4, 1.7.1, 1.7.2, 2.4, 2.5, 1.3[DZ2], 1.3[DZ3] | 24.01.2027. |
| 5 | 10 | 1.7.3, 1.7.4, 2.3, 2.6, 2.7, 2.8, 2.9, 1.4[DZ1], 1.4[DZ2] | pre odbrane |

Na kraju svake faze: svi koraci faze su u `develop`, ručni test celog toka prolazi, PR `develop → main`, tag `ocena-N`. Ako zaglavimo, poslednji tag je verzija koju branimo.

---

## 5. Plan po članovima

Svaki član ide **odozgo nadole** kroz svoj spisak: prvo lakši koraci, pa teži, a dodatni zahtevi na kraju. Korak nosi ocenu kojoj pripada i rok te faze iz sekcije 4. Ko napravi komponentu, taj je dorađuje u kasnijim koracima i prvi je objašnjava na odbrani.

**Težina** je procena obima i složenosti koraka na skali 1-5 (1 = nekoliko sati, 5 = najveći algoritamski deo projekta). Podela je napravljena tako da **zbir težina bude skoro isti za sve** i da **svako ima više teških, algoritamskih koraka** - niko ne sme da ima vidno lakši deo. Ako se na početku faze pokaže da je neki korak bitno teži ili lakši od procene, podela se koriguje PR-om na ovaj fajl.

| Član | Paketi | Teški algoritamski delovi | Ukupna težina |
|---|---|---|---|
| **Andrej** | `lru`, `blokmenadzer`, `engine`, `kes`, `iteratori`, `.github/` | Block manager, engine (read/write path), merge iterator, scan | **19.5** |
| **Aleksa** | `wal`, `memtable/skiplista`, `kodiranje`, `blob`, `mergeoperatori` | WAL osnovni i blokovski, skip lista, kompresija, BLOB, merge operatori | **20** |
| **Đole** | `sstable`, `merkle`, `lsm`, `kopije/checkpoint` | SSTable, Merkle stablo, LSM i kompakcije, MANIFEST | **20.5** |
| **Relja** | `konfiguracija`, `cmd/kvengine`, `bloomfilter`, `memtable` (interfejs, hashmapa, bstablo, menadžer), `tokenbucket`, `probabilisticki`, `kopije` (snapshot, backup) | B stablo, Bloom filter, CountMinSketch, HyperLogLog, SimHash, N Memtable instanci | **19.5** |

**Faza 0 - svi zajedno (do 22.11.):** Andrej pravi skelet (A1), ostali ga review-uju; svi kloniraju repo, prave granu `feature/<paket>-init`, prvi commit u svom paketu (npr. `doc.go`) i otvaraju PR - provera da Git tok radi kod svih.

### 5.1 Andrej - ukupno 19.5

| # | Ocena | Težina | Korak | Grana | Šta tačno | Zavisi od |
|---|---|---|---|---|---|---|
| A1 | 0 | 2 | Skelet repoa | `feature/skelet` | `go.mod`, folderi iz 3.1, interfejsi iz 3.3 (potpisi + prazna tela koja se kompajliraju), `internal/zapis` sa kodiranjem/dekodiranjem i testovima, CI (`go vet`, `go test`), `.gitignore` (`podaci/`, `checkpoint/`, `backup/`) | - |
| A2 | 6 | 1 | Generički LRU | `feature/lru` | Hash mapa + `container/list` (vežbe 10); `Uzmi`, `Stavi`, `Izbaci`; O(1) operacije | A1 |
| A3 | 6 | 3 | Block manager + block cache (1.6) | `feature/blok-menadzer` | Blokovi 4/8/16 KB iz konfiguracije; čitanje: prvo cache, pa disk, pa upis u cache; pisanje: disk + ažuriranje cache-a ako je blok tu; `PisacToka`/`CitacToka` | A2 |
| A4 | 6 | 4 | Engine: write i read path | `feature/engine-jezgro` | `Put/Get/Delete` kroz WAL → Memtable → flush u SSTable; GET kroz Memtable → SSTable-ove od najnovijeg; brisanje WAL segmenata posle flush-a (niska voda); oporavak na startu (WAL → Memtable); `Prostor` i `"default"` keyspace; zabrana `"\x00"` ključeva | A3, L1, D1, R1, R3 |
| A5 | 7 | 1 | Cache zapisa (1.5) | `feature/kes` | LRU nad zapisima posle Memtable-a; PUT/DELETE ažuriraju ili izbacuju ključ (nikad zastarela vrednost); GET sa diska upisuje u cache | A4 |
| A6 | 8 | 2.5 | Merge iterator | `feature/merge-iterator` | Spaja više `Iterator`-a (Memtable + SSTable-ovi) u jedan sortiran tok; za isti ključ pobeđuje najnoviji; osnova za kompakciju (D5) i scan (A7) | D4 |
| A7 | 9 | 2.5 | PREFIX_SCAN / RANGE_SCAN (2.4) | `feature/scan` | Preko merge iteratora; pozicioniranje preko Summary/Index na prvi kandidat ključ; paginacija (strana, veličina); preskakanje tombstone i sistemskih ključeva; **algoritam u README-u** | A6 |
| A8 | 9 | 1.5 | PREFIX_ITERATE / RANGE_ITERATE (2.5) | `feature/iteratori` | Interaktivni NEXT/STOP nad istim mehanizmom kao A7; **algoritam u README-u** | A7 |
| A9 | 10 | 2 | Keyspace-ovi (2.9) | `feature/keyspace` | `USE <keyspace>` i keyspace parametar u svim operacijama; zasebne komponente po prostoru; prostori se otkrivaju iz `podaci/` na startu | A8 |

### 5.2 Aleksa - ukupno 20

| # | Ocena | Težina | Korak | Grana | Šta tačno | Zavisi od |
|---|---|---|---|---|---|---|
| L1 | 6 | 3 | WAL osnovni (1.1) | `feature/wal` | Segmenti sa fiksnim brojem zapisa (`wal_0001.log`…); format iz 3.2; CRC provera pri čitanju; čitanje blok po blok preko Block managera; brisanje segmenata do niske vode; test oporavka posle "pada" | A1 (za testove do A3: jednostavna zamena za `BlokMenadzer`) |
| L2 | 7 | 2 | Memtable skip lista (1.2[DZ1]) | `feature/memtable-skiplista` | Implementira `Memtable` interfejs (vežbe 5); maksimalna visina iz konfiguracije | R3 |
| L3 | 8 | 3.5 | WAL blokovski (1.1[DZ1]) | `feature/wal-blokovi` | Segment fiksne dužine u blokovima; fragmentacija FULL/FIRST/MIDDLE/LAST; padding; trailing canary; `Sinhronizuj` (fdatasync na Linux-u, `File.Sync()` na Windows-u); upisati konačan format u 3.2 | L1 |
| L4 | 9 | 3 | Kompresovani format data bloka (1.3[DZ3]) | `feature/kodiranje` | Varint za sva numerička polja (`binary.PutUvarint`/`PutVarint`); delta enkoding ključeva; dictionary/bit packing gde ima smisla (**obrazloženje izbora u README-u**); zamena u SSTable-u (sa Đoletom) | D1 |
| L5 | 9 | 1.5 | Data block hash index (1.3[DZ2]) | `feature/hash-index-bloka` | Hash index unutar data bloka za brže nalaženje ključa u bloku | L4 |
| L6 | 10 | 2 | BATCH_WRITE / DELETE_RANGE (2.6) | `feature/batch` | Transakciono: ceo batch je jedan atomičan upis u WAL (`Dodaj(zapisi...)`); posle pada se oporavlja ceo batch ili ništa; DELETE_RANGE nalazi ključeve u opsegu (preko A6/A7) i upisuje tombstone-ove kao jedan batch | L3, A7 |
| L7 | 10 | 2.5 | BLOB (2.8) | `feature/blob` | Vrednost veća od `blob.prag_kb` ide u zaseban fajl (preko Block managera), a Data čuva referencu (`TipBlobReferenca`); GET razrešava referencu; kompakcija briše BLOB-ove na koje više niko ne pokazuje (sa Đoletom) | D5 |
| L8 | 10 | 2.5 | Merge operatori za BF/CMS/HLL (2.3) | `feature/merge-operatori` | Dodavanje elementa ne čita prethodno stanje, nego upisuje samo operaciju (`TipMergeOperacija`); čitanje skuplja sve operacije za taj ključ sa svih nivoa i primenjuje ih na strukturu (Reljine strukture iz R8); ponašanje kompakcije prema merge zapisima (sa Đoletom: sme da ih spoji u jedno stanje); parametri i hash funkcije u meta bloku SSTable-a (sa Đoletom) | R8, D5 |

### 5.3 Đole - ukupno 20.5

| # | Ocena | Težina | Korak | Grana | Šta tačno | Zavisi od |
|---|---|---|---|---|---|---|
| D1 | 6 | 4.5 | SSTable (1.3.1-1.3.4) | `feature/sstable` | Data, Filter (Bloom), Index, Summary sa min/max ključem, TOC (vežbe 9); `Napravi`, `Otvori`, `Nadji`; sve preko `PisacToka`/`CitacToka`, nikad ceo fajl u memoriji; kodiranje zapisa na jednom mestu (vidi 3.2) | A1, R2 (za testove do A3: jednostavna zamena za `BlokMenadzer`) |
| D2 | 7 | 0.5 | Proređenost Summary-ja (1.3[DZ1]) | `feature/summary-proredjenost` | `proredjenost_summary` iz konfiguracije (npr. 5 = svaki 5. zapis iz Indexa ide u Summary) | D1 |
| D3 | 7 | 2.5 | Merkle stablo (1.3.5) | `feature/merkle` | Stablo nad vrednostima iz Data; serijalizacija u Metadata fajl (po nivoima, vežbe 8); komanda `VALIDATE <generacija>` ponovo gradi stablo iz Data i javlja **da li i koji zapisi** su izmenjeni | D1 |
| D4 | 8 | 1.5 | SSTable iterator | `feature/sstable-iterator` | `Tabela.Iterator(odKljuca)` čita blok po blok; pozicioniranje preko Summary/Index; osnova za merge iterator (A6) i kompakcije | D2 |
| D5 | 9 | 5 | LSM stablo + size-tiered i leveled kompakcija (1.4) | `feature/lsm` | Za svaku tabelu poznat nivo; `max_nivoa` iz konfiguracije; oba algoritma sa parametrima iz konfiguracije; lančane kompakcije (nivo i može da pokrene nivo i+1); spajanje preko merge iteratora (A6); izbacivanje zastarelih verzija; tombstone se izbacuje tek na poslednjem nivou; redosled pretrage tabela na read path-u za oba algoritma | D4, A6 |
| D6 | 9 | 1 | Checkpoint (1.7.2) | `feature/checkpoint` | Flush Memtable-a, pa hard linkovi (`os.Link`) ka svim tabelama (i MANIFEST-u, kad postoji) u `checkpoint/<ime>/` | D5 |
| D7 | 10 | 1.5 | Hibridna kompakcija (1.4[DZ1]) | `feature/lsm-hibrid` | Prvih `hibridni_size_tiered_nivoa` nivoa size-tiered, ostali leveled | D5 |
| D8 | 10 | 2.5 | MANIFEST (1.4[DZ2]) | `feature/manifest` | MANIFEST beleži svaku promenu LSM stabla (nova tabela, kompakcija, brisanje); oporavak LSM stabla iz MANIFEST-a na startu; nivo se više ne čita iz imena fajla | D7 |
| D9 | 10 | 1.5 | TTL (2.7) | `feature/ttl` | TTL za jedan ključ ili grupu ključeva (polje `Istice`, `PUT ... [ttl_sekundi]`); GET/scan/iterate ne vraćaju istekle zapise (izmena u engine-u i scan-u u dogovoru sa Andrejem); kompakcija ih fizički uklanja | D5, A7 |

### 5.4 Relja - ukupno 19.5

| # | Ocena | Težina | Korak | Grana | Šta tačno | Zavisi od |
|---|---|---|---|---|---|---|
| R1 | 6 | 2.5 | Konfiguracija (2.1) + CLI | `feature/konfiguracija-cli` | Učitavanje `config.json`, default vrednosti u kodu, validacija pri pokretanju; CLI petlja sa PUT/GET/DELETE/EXIT; mehanizam da drugi lako dodaju svoje komande | A1 |
| R2 | 6 | 1.5 | Bloom filter | `feature/bloom-filter` | Parametri iz očekivanog broja elemenata i verovatnoće greške (vežbe 4); ručna serijalizacija; koristi ga D1 i tip BloomFilter (R8) | A1 |
| R3 | 6 | 1 | Memtable hash mapa (1.2) | `feature/memtable-hashmapa` | Interfejs `Memtable`; hash mapa; ograničenje po broju elemenata ili KB; `SortiraniZapisi` za flush | A1 |
| R4 | 7 | 3 | Memtable B stablo + izbor strukture (1.2[DZ1]) | `feature/memtable-bstablo` | Implementira `Memtable` (vežbe 6); red stabla iz konfiguracije; `memtable.struktura` bira hashmapu, skip listu ili B stablo | R3 |
| R5 | 8 | 1.5 | Token Bucket (2.2) | `feature/token-bucket` | Broj tokena i interval iz konfiguracije (vežbe 12); stanje (timestamp poslednjeg resetovanja + broj tokena) ručno serijalizovano kao zapis pod `"\x00sys/token_bucket"`; provera pre svake korisničke operacije; odbijen zahtev vraća poruku; korisnik ne vidi i ne menja taj zapis | A4 |
| R6 | 8 | 2 | N Memtable instanci (1.2[DZ2]) | `feature/memtable-instance` | 1 read-write + N-1 read-only; kad se RW napuni postaje RO; flush kad se popune sve; GET od najnovije ka najstarijoj; WAL segmenti se brišu tek kad je njihov sadržaj flush-ovan (sa Aleksom i Andrejem) | R4 |
| R7 | 9 | 1.5 | Snapshot (1.7.1) | `feature/snapshot` | Čitanje vrednosti pod ključem kroz vreme, samo u memoriji (tumačenje potvrditi - sekcija 8) | R6 |
| R8 | 10 | 2.5 | Probabilistički tipovi (2.3): CountMinSketch, HyperLogLog, SimHash + komande | `feature/probabilisticki` | Strukture (vežbe 4 i 5) sa ručnom serijalizacijom i funkcijom za primenu jedne operacije (koristi je Aleksin merge operator, L8); komande BF/CMS/HLL/SIMHASH iz 3.6 (kreiranje, brisanje, dodavanje, upit); čuvaju se kao sistemski zapisi; SimHash čuva fingerprint i računa Hamingovu udaljenost. **Prvi korak u fazi 10**, jer L8 zavisi od njega | R2, A4 |
| R9 | 10 | 2 | Backup (1.7.3) + RESTORE | `feature/backup` | Kopira ceo sadržaj u `backup/<ime>/` (kopiranje sadržaja preko Block managera); RESTORE iz izabranog backup-a | D6 |
| R10 | 10 | 2 | Inkrementalni backup (1.7.4) | `feature/backup-inkrementalni` | Kopira samo fajlove nastale/izmenjene posle prethodnog backup-a (SSTable-ovi su nepromenljivi, pa je dovoljno pratiti koje generacije su već kopirane); RESTORE iz lanca backup-a | R9, D8 |

### 5.5 Pregled: težina po fazama

| Faza | Andrej | Aleksa | Đole | Relja |
|---|---|---|---|---|
| 0 | A1 (2) | init | init | init |
| 6 | A2, A3, A4 (8) | L1 (3) | D1 (4.5) | R1, R2, R3 (5) |
| 7 | A5 (1) | L2 (2) | D2, D3 (3) | R4 (3) |
| 8 | A6 (2.5) | L3 (3.5) | D4 (1.5) | R5, R6 (3.5) |
| 9 | A7, A8 (4) | L4, L5 (4.5) | D5, D6 (6) | R7 (1.5) |
| 10 | A9 (2) | L6, L7, L8 (7) | D7, D8, D9 (5.5) | R8, R9, R10 (6.5) |
| **Ukupno** | **19.5** | **20** | **20.5** | **19.5** |

Zbir je skoro isti; po fazama ima razlika (npr. Andrej je najopterećeniji u fazi 6 jer engine spaja sve komponente). Ko u nekoj fazi završi ranije, kreće na sledeći korak sa svog spiska čim su mu zavisnosti spremne - ne čeka kraj faze.

### 5.6 Rotacija review-a

| Autor PR-a | Reviewer |
|---|---|
| Andrej | Đole |
| Đole | Aleksa |
| Aleksa | Relja |
| Relja | Andrej |

Rotacija se pomera za jedno mesto na početku svake faze, da bi svako tokom projekta pročitao kod svih ostalih.

### 5.7 Mesta gde se paketi dodiruju (obavezna koordinacija)

| Tema | Ko se dogovara |
|---|---|
| Format SSTable zapisa i kompresija | Aleksa ↔ Đole |
| Kompakcija ↔ TTL, BLOB, merge operatori, meta blok | Đole ↔ Aleksa |
| Probabilističke strukture ↔ merge operatori | Relja ↔ Aleksa |
| TTL u read path-u i scan-u | Đole ↔ Andrej |
| Memtable flush ↔ brisanje WAL segmenata | Relja ↔ Aleksa ↔ Andrej |
| Nove CLI komande | vlasnik komande ↔ Relja |
| Checkpoint/backup ↔ spisak fajlova i MANIFEST | Relja ↔ Đole |

---

## 6. Status (ažurira se kroz PR-ove)

Oznake: `[ ]` nije počelo · `[~]` u toku · `[x]` u `develop`

- **Faza 0:** [ ] A1 · [ ] svi: init grane
- **Ocena 6:** [ ] A2 · [ ] A3 · [ ] A4 · [ ] L1 · [ ] D1 · [ ] R1 · [ ] R2 · [ ] R3 · [ ] tag `ocena-6`
- **Ocena 7:** [ ] A5 · [ ] L2 · [ ] D2 · [ ] D3 · [ ] R4 · [ ] tag `ocena-7`
- **Ocena 8:** [ ] A6 · [ ] L3 · [ ] D4 · [ ] R5 · [ ] R6 · [ ] tag `ocena-8`
- **Ocena 9:** [ ] A7 · [ ] A8 · [ ] L4 · [ ] L5 · [ ] D5 · [ ] D6 · [ ] R7 · [ ] tag `ocena-9`
- **Ocena 10:** [ ] A9 · [ ] L6 · [ ] L7 · [ ] L8 · [ ] D7 · [ ] D8 · [ ] D9 · [ ] R8 · [ ] R9 · [ ] R10 · [ ] tag `ocena-10`

---

## 7. Definicija "gotovo" za svaki korak

1. Implementirano sve iz kolone "Šta tačno" i odgovarajuće stavke specifikacije.
2. Testovi za glavne i granične slučajeve (prazan ulaz, zapis tačno na granici bloka, zapis veći od bloka, oštećen CRC, ključ koji ne postoji…).
3. `go vet ./...` i `go test ./...` prolaze lokalno i u CI-ju.
4. Poštovana pravila projekta iz sekcije 0 (biblioteke, Block manager, blok po blok).
5. Nova podešavanja dodata u `config.json` i default vrednosti.
6. PR odobrio reviewer iz rotacije.
7. Vlasnik ume da objasni svaku liniju i da je izmeni uživo.
8. Status u sekciji 6 ažuriran.

---

## 8. Otvorena pitanja (proveriti sa asistentom)

1. **Greška u specifikaciji:** u 1.3 oznaka `DZ3` se pojavljuje dvaput. Pretpostavljamo da je "data block hash index" zapravo **1.3[DZ2]**. Radimo oba, pa ne menja posao, ali treba znati za odbranu.
2. **Snapshot (1.7.1):** šta tačno znači "čitanje vrednosti pod ključem kroz vreme" - sve verzije od pokretanja sistema, ili vrednosti u trenucima kada je korisnik eksplicitno napravio snapshot?
3. Da li je SSTable u **više fajlova** (po jedan za svaki element, kao na vežbama 9) prihvatljiv, ili se očekuje jedan fajl?
4. Block cache pri pisanju: specifikacija kaže "piše se na disk i ažurira cache ako je blok tu", a vežbe 7 opisuju upis u buffer pool pa kasnije na disk. **Radimo po specifikaciji** - potvrditi.
5. Datum odbrane u februarskom roku.

---

## 9. Priprema za odbranu

- Na odbrani **svako** dobija eliminacioni zadatak koji menja projekat. Zato svako mora da poznaje ceo sistem, a ne samo svoj deo.
- Na kraju svake faze: kratak sastanak na kom svaki vlasnik objasni svoje komponente ostalima (10-15 min po osobi).
- Tipične izmene za vežbu: promeni format zapisa (dodaj polje), promeni uslov flush-a, dodaj novu komandu u CLI, promeni algoritam izbacivanja u cache-u, promeni redosled provere na read path-u, promeni uslov kompakcije, dodaj novi parametar u konfiguraciju.
- Algoritmi koje specifikacija traži da se obrazlože (scan, iteratori, izbor kompresije) moraju biti zapisani u `README.md`.

---

## 10. Početni prompt za agenta (kopiraj na početku sesije)

```
Radim na timskom projektu Key-Value engine (NASP), cilj je ocena 10. Pročitaj ceo implementation
plan u root-u repoa - posebno sekciju 0 (način rada i pravila projekta) i moj spisak koraka u sekciji 5.
Ja sam <IME>. Radim na koraku <ID> ("<naziv>") na grani <feature/...>.
Ti si mentor, ne autor: ne pravi i ne menjaj fajlove. Ja ću ti reći svoju ideju, ti mi daj kritičko
mišljenje i šta specifikacija traži, pa zajedno dođemo do rešenja. Kad se dogovorimo, daj mi kod u
blokovima (fajl po fajl, sa putanjom) i objasni sintaksu i logiku - ja ću ga ručno prekucati.
```
