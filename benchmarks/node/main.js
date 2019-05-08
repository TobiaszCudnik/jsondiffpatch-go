const fs = require('fs')
const jsondiffpatch = require('jsondiffpatch')
const microtime = require('microtime')

const TRIES = 100000

const left_text = fs.readFileSync('./test/fixtures/1/left.json')
const right_text = fs.readFileSync('./test/fixtures/1/right.json')

const left = JSON.parse(left_text)
const right = JSON.parse(right_text)
const differ = new jsondiffpatch.DiffPatcher()


const start = microtime.now()

// benchmark
for (let i = 0; i <= TRIES; i++) {
    // make the diff
    differ.diff(left, right)
}

const elapsed = microtime.now() - start

console.log(`Tries: ${TRIES}`)
console.log(`Time: ${elapsed} (micro secs)`)
