'use strict';

var babel = require('gulp-babel');
var rename = require("gulp-rename");

function Task(lib, gulp) {
  var scripts = lib.scripts,
    plugins = lib.plugins;

  gulp.task('release.js', ['app'], function () {
    return gulp
      .src(scripts.qorAdmin)
      .pipe(plugins.concat('qor_admin_default.js'))
      .pipe(gulp.dest(scripts.dest));
  });
  gulp.task('release.js+', ['release.js'], function () {
    return gulp
      .src(lib.pathto('javascripts/qor_admin_default.js'))
      .pipe(rename({suffix: ".min"}))
      .pipe(babel({
        presets: ['es2015']
      }))
      .pipe(plugins.uglify())
      .pipe(gulp.dest(scripts.dest));
  });

}

exports.Task = Task;
