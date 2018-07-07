# wordgraph

![screencast](https://raw.githubusercontent.com/jackdoe/wordgraph/master/tty.gif)

Build a graph of flow of words like "a -> b -> c -> d"

I trained it on 15000 books from project gutenberg, and cutoff the
edges that are below 85th percentile. (you can find this index in computed_index/)

the thing is also deployed on https://wordgraph.xyz and you can try it at https://wordgraph.xyz/test

## build an index

```
$ go run *.go -parseRoot ./txt
[word count]
 100% |████████████████████████████████████████| [1m20s:0s]
[create graph]
   8% |███                                     | [30s:5m48s]
cleaning up the graph nodes: 152960, edges before: 17467743, after: 2949628
  17% |██████                                  | [1m4s:5m16s]
cleaning up the graph nodes: 179988, edges before: 17088474, after: 3274320
  26% |██████████                              | [1m37s:4m37s]
cleaning up the graph nodes: 194081, edges before: 17784098, after: 3665601
  35% |██████████████                          | [2m15s:4m12s]
cleaning up the graph nodes: 200530, edges before: 19089495, after: 4125064
  44% |█████████████████                       | [2m51s:3m37s]
cleaning up the graph nodes: 205077, edges before: 17764668, after: 3948223
  49% |███████████████████                     | [3m18s:3m26s]

...

```

In txt/ i have the books from project gutenberg downloaded as `wget -w 2 -m -H "http://www.gutenberg.org/robot/harvest?filetypes[]=txt&langs[]=en"`
(you can read more how to get the books from https://www.gutenberg.org/wiki/Gutenberg:Information_About_Robot_Access_to_our_Pages)

The books are stored in .zip files, and we extract the .txt files on the fly, no need to extract the zip files

`-parseRoot txt/` is looking for .zip or .txt files in the txt/ directory


## /flow

for example if you look at 'this is an apple' vs 'this is a apple':

```
$ curl -s -d '{"text":"this is a apple"}' https://wordgraph.xyz/flow | json_pp
{
   "Items" : [
      {
         "Word" : "this",
         "Score" : 2
      },
      {
         "Word" : "is",
         "Score" : 3
      },
      {
         "Score" : 3,
         "Word" : "a"
      },
      {
         "Score" : 0,
         "Word" : "apple"
      }
   ]
}

$ curl -s -d '{"text":"this is an apple"}' https://wordgraph.xyz/flow | json_pp
{
   "Items" : [
      {
         "Word" : "this",
         "Score" : 2
      },
      {
         "Score" : 4,
         "Word" : "is"
      },
      {
         "Word" : "an",
         "Score" : 4
      },
      {
         "Score" : 3,
         "Word" : "apple"
      }
   ]
}

```

## /query

```

$ curl -d '{"text":"apple","percentile":80}' https://wordgraph.xyz/query | json_pp
{
   "Tokenized" : [
      "apple"
   ],
   "Processed" : {
      "apple" : {
         "Word" : "apple",
         "Flows" : [
            {
               "Words" : [
                  "the",
                  "apple",
                  "tree"
               ],
               "Count" : 699
            },
            {
               "Words" : [
                  "the",
                  "apple",
                  "of"
               ],
               "Count" : 456
            },
            {
               "Count" : 372,
               "Words" : [
                  "an",
                  "apple",
                  "tree"
               ]
            },
            {
               "Words" : [
                  "an",
                  "apple",
                  ","
               ],
               "Count" : 328
            },
            {
               "Count" : 285,
               "Words" : [
                  "the",
                  "apple",
                  "trees"
               ]
            },
            {
               "Count" : 275,
               "Words" : [
                  "the",
                  "apple",
                  ","
         ...
```

## FIXME

* the tokenizer is not good
* add part-of-speech tagging and use that as alternative graph

## LICENSE

MIT
