'use strict';

var plumber = require('gulp-plumber');

function Task(lib, gulp) {
  var styles = lib.styles,
    plugins = lib.plugins;

  gulp.task('sass', function () {
    return gulp
      .src(styles.src)
      .pipe(plumber())
      .pipe(plugins.sass())
      .pipe(gulp.dest(styles.dest));
  });
}

exports.Task = Task;
