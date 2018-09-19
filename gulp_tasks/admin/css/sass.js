'use strict';

var plumber = require('gulp-plumber');

function Task(lib, gulp) {
  let styles = lib.styles,
    plugins = lib.plugins;

  gulp.task('sass', function () {
    return gulp
      .src(styles.src)
      .pipe(plumber())
      .pipe(plugins.sass().on('error', plugins.sass.logError))
      .pipe(gulp.dest(styles.dest));
  });
}

exports.Task = Task;
