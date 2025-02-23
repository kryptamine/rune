
fun makeCounter() {
  var i = 0;
  fun count() {
    i = i + 2;

    return i;
  }
  return count;
}

var counter = makeCounter();
counter();
counter();
print(counter());

