const fs = require('fs')
const jsondiffpatch = require('jsondiffpatch')
const microtime = require('microtime')

const TRIES = 100
// const TRIES = 1

const left_text = fs.readFileSync('./test/fixtures/2/left.json')
const right_text = fs.readFileSync('./test/fixtures/2/right.json')

const left = JSON.parse(left_text)
const right = JSON.parse(right_text)

// const left = []
// const right = []
//
// prepare a separate fixture per test
// for (let i = 0; i <= TRIES; i++) {
//     left[i] = JSON.parse(left_text)
//     right[i] = JSON.parse(right_text)
// }

const differ = new jsondiffpatch.DiffPatcher()

// warm up the JIT
differ.diff([1,2,3,4], [1,3,4])
differ.diff([1,2,3,4], [5,6,6])
differ.diff([1,2,3,4], [9,0,1])

const start = microtime.now()

// benchmark
for (let i = 0; i < TRIES; i++) {
    // make the diff
    // differ.diff(left[i], right[i])
    differ.diff(left, right)
}

const elapsed = microtime.now() - start

console.log(`Tries: ${TRIES}`)
console.log(`Time: ${elapsed} (micro secs)`)
