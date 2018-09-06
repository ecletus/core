'use strict';

var plumber = require('gulp-plumber');

function Task(lib, gulp) {
  var styles = lib.styles,
    plugins = lib.plugins;

  gulp.task('css', ['sass'], function () {
    return gulp
      .src(styles.main)
      .pipe(plumber())
      .pipe(plugins.autoprefixer())
      .pipe(plugins.csscomb())
      .pipe(plugins.minifyCss())
      .pipe(gulp.dest(styles.dest));
  });
}

exports.Task = Task;
