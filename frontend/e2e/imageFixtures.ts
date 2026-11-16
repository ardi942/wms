import { Buffer } from 'node:buffer'

export interface ImageFile {
  name: string
  mimeType: string
  buffer: Buffer
}

export const tinyPng: ImageFile = {
  name: 'dummy.png',
  mimeType: 'image/png',
  buffer: Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAAAMAAAADCAYAAABWKLW/AAAAF0lEQVR4nGL6z8DwnwEbYELmAAIAAP//SUYCA3sJO5kAAAAASUVORK5CYII=',
    'base64',
  ),
}

export const tinyJpeg: ImageFile = {
  name: 'dummy.jpg',
  mimeType: 'image/jpeg',
  buffer: Buffer.from(
    '/9j/2wCEAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDIBCQkJDAsMGA0NGDIhHCEyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMv/AABEIAAMAAwMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/APAZJJJpXlldnkdizOxyWJ6knuabRRQCVj//2Q==',
    'base64',
  ),
}

export const oversizedPng: ImageFile = {
  name: 'oversized.png',
  mimeType: 'image/png',
  buffer: Buffer.alloc(2 * 1024 * 1024 + 1),
}

export const invalidTextFile: ImageFile = {
  name: 'not-an-image.txt',
  mimeType: 'text/plain',
  buffer: Buffer.from('not an image'),
}
