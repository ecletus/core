'use strict';

var babel = require('gulp-babel'),
  eslint = require('gulp-eslint'),
  plumber = require('gulp-plumber');

function Task(lib, gulp) {
  var scripts = lib.scripts,
    plugins = lib.plugins;

  gulp.task('app', ['qor'], function () {
    return gulp
      .src(scripts.src)
      .pipe(plumber())
      .pipe(plugins.concat('app.js'))
      .pipe(gulp.dest(scripts.dest));
  });

  gulp.task('app+', function () {
    return gulp
      .src(scripts.src)
      .pipe(plumber())
      .pipe(
        eslint({
          configFile: '.eslintrc'
        })
      )
      .pipe(
        babel({
          presets: ['es2015']
        })
      )
      .pipe(eslint.format())
      .pipe(plugins.concat('app.min.js'))
      .pipe(plugins.uglify())
      .pipe(gulp.dest(scripts.dest));
  });
}

exports.Task = Task;
