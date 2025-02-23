var a = [1, 10, 5, 3, 1];

print("Array before sorting:");
print(a);

fun bubbleSort(arr) {
  var length = len(arr);

  for (var i = 0; i < length; i = i + 1) {
    for (var j = 0; j < length - i - 1; j = j + 1) {
      if (arr[j] > arr[j + 1]) {
        var temp = arr[j];
        arr[j] = arr[j + 1];
        arr[j + 1] = temp;
      }
    }
  }

  return arr;
}

print("Array after sorting:");
print(bubbleSort(a));
