10 REM --- Super Talker II (Corrected) ---
20 REM Enhancements: More words, more templates, avoids immediate sentence repetition.
30 CLS
40 PRINT "Super Talker II"
50 PRINT "Generates more varied sentences."
60 PRINT "Press CTRL-C to stop."
70 PRINT

80 REM --- Seed Random Number Generator ---
90 RESTORE

100 REM --- Define Word Category Ranges (inclusive) ---
110 REM V = Verb, N = Noun, A = Adjective, ADV = Adverb, P = Preposition
120 REM --- Counts ---       --- Ranges ---
130 REM Verbs:       50      (1-50)
140 REM Nouns:       70      (51-120) - Objects, concepts, places, animals
150 REM Adjectives:  50      (121-170)
160 REM Adverbs:     30      (171-200)
170 REM Prepositions: 20     (201-220)
180 REM TOTALWORDS = 220
190 REM NUMTEMPLATES = 9 

200 REM --- Dimension Array & Variables ---
210 DIM W$(220)
220 LASTS$ = "" : REM Stores the last generated sentence
230 LASTT = 0 : REM Stores the last used template number

240 REM --- Read Word Data into Array W$ ---
250 READ INFO$ : REM Read and discard the first "CATEGORY INFO" data item
260 FOR I=1 TO 220
262 READ W$(I)
263 NEXT I
270 IF I <> 221 THEN PRINT "Error reading data, expected 220 words, read ";I-1 : END
280 REM --- Main Loop ---
290 PRINT : PRINT "---" : REM Separator and blank line
300 REM --- Generate Sentence Attempt ---
310 GOSUB 9500 : REM Generate sentence into S$, template choice into T
320 REM --- Check for immediate repetition ---
330 IF S$ = LASTS$ THEN PRINT "(Retrying - same sentence)" : GOTO 310
340 REM --- Update last sentence/template and output ---
350 LASTS$ = S$
360 LASTT = T
370 GOSUB 9900 : REM Output the sentence S$ (MOVED TO 9900)
380 GOTO 290 : REM Loop for next sentence

999 REM --- End of Main Program Logic ---

1000 REM --- Template 1: I [ADV] VERB the [ADJ] NOUN [PREP the NOUN]. ---
1010 V = RND(50)+1     : REM Verb (1-50)
1020 N1 = RND(70)+51    : REM Noun 1 (51-120)
1030 S$ = "I "
1040 IF RND(0) > 0.6 THEN ADV = RND(30)+171 : S$ = S$ + W$(ADV) + " " : REM Optional Adverb
1050 S$ = S$ + W$(V) + " the "
1060 IF RND(0) > 0.4 THEN A = RND(50)+121 : S$ = S$ + W$(A) + " " : REM Optional Adjective
1070 S$ = S$ + W$(N1)
1080 IF RND(0) > 0.7 THEN P = RND(20)+201 : N2 = RND(70)+51 : S$ = S$ + " " + W$(P) + " the " + W$(N2) : REM Optional Prep. Phrase
1090 S$ = S$ + "."
1100 RETURN

2000 REM --- Template 2: The ADJ NOUN [ADV] VERB PREP the ADJ NOUN. ---
2010 A1 = RND(50)+121   : REM Adjective 1 (121-170)
2020 N1 = RND(70)+51    : REM Noun 1 (51-120)
2030 V = RND(50)+1      : REM Verb (1-50)
2040 P = RND(20)+201    : REM Preposition (201-220)
2050 A2 = RND(50)+121   : REM Adjective 2 (121-170)
2060 N2 = RND(70)+51    : REM Noun 2 (51-120)
2070 S$ = "The " + W$(A1) + " " + W$(N1) + " "
2080 IF RND(0) > 0.5 THEN ADV = RND(30)+171 : S$ = S$ + W$(ADV) + " " : REM Optional Adverb
2090 S$ = S$ + W$(V) + " " + W$(P) + " the " + W$(A2) + " " + W$(N2) + "."
2100 RETURN

3000 REM --- Template 3: Sometimes/Often/Maybe the NOUN VERB [ADV] [PREP the NOUN]. ---
3010 RADVSTART = RND(3) : REM Choose starting adverb (0, 1, or 2)
3020 IF RADVSTART = 0 THEN SADV$ = "Sometimes "
3030 IF RADVSTART = 1 THEN SADV$ = "Often "
3040 IF RADVSTART = 2 THEN SADV$ = "Maybe "
3050 N1 = RND(70)+51    : REM Noun 1 (51-120)
3060 V = RND(50)+1      : REM Verb (1-50)
3070 S$ = SADV$ + "the " + W$(N1) + " " + W$(V)
3080 IF RND(0) > 0.4 THEN ADV = RND(30)+171 : S$ = S$ + " " + W$(ADV) : REM Optional Adverb
3090 IF RND(0) > 0.6 THEN P = RND(20)+201 : N2 = RND(70)+51 : S$ = S$ + " " + W$(P) + " the " + W$(N2) : REM Optional Prep. Phrase
3100 S$ = S$ + "."
3110 RETURN

4000 REM --- Template 4: Why is the ADJ NOUN VERB [ADV]? ---
4010 A = RND(50)+121    : REM Adjective (121-170)
4020 N = RND(70)+51     : REM Noun (51-120)
4030 V = RND(50)+1      : REM Verb (1-50)
4040 S$ = "Why is the " + W$(A) + " " + W$(N) + " " + W$(V)
4050 IF RND(0) > 0.7 THEN ADV = RND(30)+171 : S$ = S$ + " " + W$(ADV) : REM Optional Adverb (less likely)
4060 S$ = S$ + "?"
4070 RETURN

5000 REM --- Template 5: People [ADV] VERB the ADJ and ADJ NOUN. ---
5010 V = RND(50)+1      : REM Verb (1-50)
5020 A1 = RND(50)+121   : REM Adjective 1 (121-170)
5030 A2 = RND(50)+121   : REM Adjective 2 (121-170)
5040 N = RND(70)+51     : REM Noun (51-120)
5050 S$ = "People "
5060 IF RND(0) > 0.5 THEN ADV = RND(30)+171 : S$ = S$ + W$(ADV) + " " : REM Optional Adverb
5070 S$ = S$ + W$(V) + " the " + W$(A1) + " and " + W$(A2) + " " + W$(N) + "."
5080 RETURN

6000 REM --- Template 6: The [ADJ] NOUN VERB [ADV]. (Simpler) ---
6010 N = RND(70)+51     : REM Noun (51-120)
6020 V = RND(50)+1      : REM Verb (1-50)
6030 S$ = "The "
6040 IF RND(0) > 0.3 THEN A = RND(50)+121 : S$ = S$ + W$(A) + " " : REM Optional Adjective (more likely)
6050 S$ = S$ + W$(N) + " " + W$(V)
6060 IF RND(0) > 0.6 THEN ADV = RND(30)+171 : S$ = S$ + " " + W$(ADV) : REM Optional Adverb
6070 S$ = S$ + "."
6080 RETURN

7000 REM --- Template 7: Think about the ADJ NOUN PREP the NOUN. ---
7010 A = RND(50)+121    : REM Adjective (121-170)
7020 N1 = RND(70)+51    : REM Noun 1 (51-120)
7030 P = RND(20)+201    : REM Preposition (201-220)
7040 N2 = RND(70)+51    : REM Noun 2 (51-120)
7050 S$ = "Think about the " + W$(A) + " " + W$(N1) + " " + W$(P) + " the " + W$(N2) + "."
7060 RETURN

8000 REM --- Template 8: The NOUN that VERB is [ADV] ADJ. ---
8010 N = RND(70)+51     : REM Noun (51-120)
8020 V = RND(50)+1      : REM Verb (1-50)
8030 A = RND(50)+121    : REM Adjective (121-170)
8040 S$ = "The " + W$(N) + " that " + W$(V) + " is "
8050 IF RND(0) > 0.5 THEN ADV = RND(30)+171 : S$ = S$ + W$(ADV) + " " : REM Optional Adverb
8060 S$ = S$ + W$(A) + "."
8070 RETURN

9000 REM --- Template 9: Is it true that people VERB PREP ADJ things? ---
9010 V = RND(50)+1      : REM Verb (1-50)
9020 P = RND(20)+201    : REM Preposition (201-220)
9030 A = RND(50)+121    : REM Adjective (121-170)
9040 S$ = "Is it true that people " + W$(V) + " " + W$(P) + " " + W$(A) + " things?"
9050 RETURN

9500 REM --- Subroutine: Generate Sentence into S$, Template into T ---
9510 REM Try to pick a template different from the last one
9520 T = RND(9)+1 : REM Choose a template (1 to 9)
9530 IF T = LASTT THEN T = RND(9)+1 : REM Simple retry if same template
9540 REM --- Select Template Subroutine Based on T ---
9550 IF T=1 THEN GOSUB 1000
9560 IF T=2 THEN GOSUB 2000
9570 IF T=3 THEN GOSUB 3000
9580 IF T=4 THEN GOSUB 4000
9590 IF T=5 THEN GOSUB 5000
9600 IF T=6 THEN GOSUB 6000
9610 IF T=7 THEN GOSUB 7000
9620 IF T=8 THEN GOSUB 8000
9630 IF T=9 THEN GOSUB 9000 : REM *** CORRECTED: Calls Template 9 now ***
9640 RETURN

9900 REM --- Subroutine: Output Sentence (MOVED HERE) ---
9910 PRINT S$ : REM Print to console
9920 SAY S$, WAIT : REM Use SAY,WAIT to wait for the speech to finish
9930 WAIT 1000 : REM Additional wait in milliseconds if needed
9940 RETURN

10000 REM --- DATA Section ---
10010 REM First item is just info, read and discarded in line 250
10020 DATA "CATEGORY INFO: V(1-50), N(51-120), A(121-170), ADV(171-200), P(201-220)"

10100 REM Verbs (1-50) - Increased count
10110 DATA see,find,make,like,know,think,feel,need,try,help,call,keep,let,show,hear,play,run,walk,write,read,learn,move,work,wait,hope,believe,remember,forget,understand,explain,ask,tell,give,take,use,want,become,leave,stay,watch,follow,lead,meet,join,save,lose,build,break,choose,change

10200 REM Nouns (51-120) - Increased count and variety
10210 DATA people,things,words,stories,questions,answers,problems,ideas,books,letters,days,nights,time,life,world,home,town,city,country,road,car,tree,flower,food,water,air,light,sound,voice,face,hand,eye,foot,head,heart,mind,dream,plan,goal,memory,friend,enemy,child,parent,teacher,student,dog,cat,bird,fish,sun,moon,star,sky,cloud,rain,snow,wind,fire,earth,game,job,art,music,color,shape,number,reason,truth,lie

10300 REM Adjectives (121-170) - Increased count
10310 DATA big,small,old,new,young,good,bad,fast,slow,smart,kind,brave,strong,weak,rich,poor,quiet,loud,funny,serious,bright,dark,soft,hard,clean,dirty,empty,full,early,late,hot,cold,wet,dry,sharp,round,flat,deep,high,low,short,tall,wide,narrow,thick,thin,clear,cloudy,calm,busy

10400 REM Adverbs (171-200) - Increased count
10410 DATA quickly,slowly,happily,sadly,loudly,quietly,carefully,always,never,often,sometimes,usually,really,very,almost,rarely,maybe,perhaps,certainly,clearly,well,badly,truly,fully,nearly,suddenly,finally,actually,especially,simply

10500 REM Prepositions (201-220) - Increased count
10510 DATA with,about,for,from,in,on,under,over,behind,beside,near,far,up,down,through,across,against,among,around,without

99999 END