'use strict';

function Task(lib, gulp) {
  gulp.task('vendors', ['vendors.js', 'vendors.css']);
}

exports.Task = Task;
