'use strict';

function Task(lib, gulp) {
  gulp.task('release', ['release.js', 'release.css']);
  gulp.task('release+', ['release.js+', 'release.css+']);
}

exports.Task = Task;
