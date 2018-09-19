'use strict';

var cleanCSS = require('gulp-clean-css'),
  sourcemaps = require('gulp-sourcemaps'),
  sort = require('gulp-sort');

function Task(lib, gulp) {
  var styles = lib.styles,
    plugins = lib.plugins;

  gulp.task('release.css', ['css'], function () {
    return gulp
      .src(styles.qorAdmin)
      .pipe(plugins.concat('qor_admin_default.css'))
      .pipe(gulp.dest(styles.dest));
  });

  gulp.task('release.css+', ['css'], function () {
    return gulp
      .src(styles.qorAdmin)
      .pipe(sourcemaps.init())
      .pipe(cleanCSS({debug: true}, (details) => {
        console.info(`[gulp-clean-css] ${details.name}: ${details.stats.originalSize}`);
        console.info(`[gulp-clean-css] ${details.name}: ${details.stats.minifiedSize}`);
      }))
      .pipe(plugins.concat('qor_admin_default.css'))
      .pipe(sourcemaps.write('.', {
        addComment:false
      }))
      .pipe(gulp.dest(styles.dest));
  });
}

exports.Task = Task;
