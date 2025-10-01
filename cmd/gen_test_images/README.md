# Test Image Generator

This command generates and uploads test images for all test ads (ads with `user_id = 1`) to B2 storage.

## Features

- **Simple Image Generation**: Creates images with a random background color and centered text showing:
  - Ad ID
  - Image number
  - Image dimensions
- **Multiple Sizes**: Generates images in three sizes (160w, 480w, 1200w) for responsive display
- **WebP Format**: Uses WebP encoding for optimal compression and faster uploads
- **Batch Upload**: Uploads all images to B2 in parallel batches for efficiency
- **Progress Tracking**: Shows upload progress every 50 successful uploads

## Usage

```bash
cd cmd/gen_test_images
go run main.go
```

## Requirements

- B2 credentials must be set in environment variables:
  - `B2_MASTER_KEY_ID`
  - `B2_KEY_ID` 
  - `B2_APP_KEY`
  - `B2_BUCKET_NAME`

## Image Structure

Images are organized in B2 as:
```
{ad_id}/{image_number}-{size}.webp
```

For example:
- `23/1-160w.webp` - Ad 23, Image 1, 160px width
- `23/1-480w.webp` - Ad 23, Image 1, 480px width  
- `23/1-1200w.webp` - Ad 23, Image 1, 1200px width
- `23/2-160w.webp` - Ad 23, Image 2, 160px width
- etc.

## Performance

- Processes all test ads in the database
- Generates 1-5 images per ad (based on `image_count` field)
- Creates 3 sizes per image (160w, 480w, 1200w)
- Uploads in batches of 10 concurrent uploads
- Uses WebP compression for optimal file sizes

## Example Output

```
Found 1000 test ads to process
Generated 15000 images to upload
Starting batch upload of 15000 images with batch size 10
Uploaded 50/15000 images
Uploaded 100/15000 images
...
Upload complete: 15000 successful, 0 failed
Test image generation and upload complete!
```
