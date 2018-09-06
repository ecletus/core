'use strict';

var babel = require('gulp-babel'),
  eslint = require('gulp-eslint'),
  plumber = require('gulp-plumber');

function Task(lib, gulp) {
  var scripts = lib.scripts,
    plugins = lib.plugins,
    basic = function() {
      return gulp.src([scripts.qorInit, scripts.qorCommon, scripts.qor])
        .pipe(plumber());
    };

  gulp.task('qor', function () {
    return basic()
      .pipe(plugins.concat('qor.js'))
      .pipe(gulp.dest(scripts.dest));
  });

  gulp.task('qor+', function () {
    return basic()
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
      .pipe(plugins.concat('qor.min.js'))
      .pipe(plugins.uglify())
      .pipe(gulp.dest(scripts.dest));
  });
}

exports.Task = Task;
